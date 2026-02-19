package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
	"sort"

	"github.com/Tanner-Denti/chirpy/internal/auth"
	"github.com/Tanner-Denti/chirpy/internal/database"
	"github.com/google/uuid"
)

type chirpResponse struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserId    uuid.UUID `json:"user_id"`
}

type chirpsResponse []chirpResponse

const (
	CHIRP_ID_PATH = "chirpID"
)

func (cfg *apiConfig) handlerDeleteChirp(w http.ResponseWriter, r *http.Request) {
	jwtToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized, no refresh token provided or invalid format, expected: Bearer <token>", err)
		return
	}

	jwtUserID, err := auth.ValidateJWT(jwtToken, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", err)
		return
	}

	chirpIDString := r.PathValue(CHIRP_ID_PATH)
	if chirpIDString == "" {
		respondWithError(w, http.StatusNotFound, "No empty chirp values ''", nil)
		return
	}

	chirpID, err := uuid.Parse(chirpIDString)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error", err)
		return
	}

	dbChirp, err := cfg.db.GetChirpByID(r.Context(), chirpID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error", err)
		return
	}

	if dbChirp.UserID != jwtUserID {
		respondWithError(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	err = cfg.db.DeleteChirpByID(r.Context(), dbChirp.ID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Not found", err)
	}

	respondWithJSON(w, http.StatusNoContent, struct{}{})
}

func (cfg *apiConfig) handlerGetChirpByID(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	chirpID := r.PathValue(CHIRP_ID_PATH)
	if chirpID == "" {
		respondWithError(w, http.StatusInternalServerError, "Internal server error", nil)
		return
	}

	chirpUUID, err := uuid.Parse(chirpID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID format, cannot parse into UUID", err)
		return
	}

	dbChirp, err := cfg.db.GetChirpByID(r.Context(), chirpUUID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Resource not found", err)
		return
	}

	response := chirpResponse{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserId:    dbChirp.UserID,
	}

	respondWithJSON(w, http.StatusOK, response)
}

func getAuthorIDFromRequest(r *http.Request) (uuid.UUID, error) {
	authorIDParam := r.URL.Query().Get("author_id")
	if authorIDParam == "" {
		return uuid.UUID{}, nil
	}
	authorID, err := uuid.Parse(authorIDParam)
	if err != nil {
		return uuid.UUID{}, err
	}
	return authorID, nil
}

func (cfg *apiConfig) handlerGetAllChirps(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	authorID, err := getAuthorIDFromRequest(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid author ID", err)
		return
	}
	
	var dbChirps []database.Chirp
	if authorID == (uuid.UUID{}) {
		dbChirps, err = cfg.db.GetAllChirps(r.Context())
	} else {
		dbChirps, err = cfg.db.GetChirpsByAuthorID(r.Context(), authorID)
	}
	
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database error", err)
		return
	}

	sortParam := r.URL.Query().Get("sort")
	if sortParam != "asc" && sortParam != "desc" {
		sortParam = "asc"
	}

	if sortParam == "asc" {
		sort.Slice(dbChirps, func(i int, j int) bool { return dbChirps[i].CreatedAt.Before(dbChirps[j].CreatedAt) })
	} else {
		sort.Slice(dbChirps, func(i int, j int) bool { return dbChirps[i].CreatedAt.After(dbChirps[j].CreatedAt) })
	}

	chirps := chirpsResponse{}
	for _, dbChirp := range dbChirps {
		chirps = append(chirps, chirpResponse{
			ID:        dbChirp.ID,
			CreatedAt: dbChirp.CreatedAt,
			UpdatedAt: dbChirp.UpdatedAt,
			Body:      dbChirp.Body,
			UserId:    dbChirp.UserID,
		})
	}

	respondWithJSON(w, http.StatusOK, chirps)
}

func (cfg *apiConfig) handlerCreateChirp(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	type requestBody struct {
		Body string `json:"body"`
	}

	reqBearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("TOKEN: %v, ERROR: %v\n", reqBearerToken, err)
		respondWithError(w, http.StatusBadRequest, "Bad Authorization header", err)
		return
	}

	tokenUserID, err := auth.ValidateJWT(reqBearerToken, cfg.jwtSecret)
	if err != nil {
		log.Printf("TOKEN: %v, USER: %v, ERROR: %v\n", reqBearerToken, tokenUserID, err)
		respondWithError(w, http.StatusUnauthorized, "JWT token header + claims do not match signature", err)
		return
	}

	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not read request", err)
		return
	}

	var req requestBody
	err = json.Unmarshal(reqBody, &req)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not unmarshal parameters", err)
		return
	}

	if len(req.Body) > 140 {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long", nil)
		return
	}

	cleanedBody := replaceBadWords(req.Body)

	chirpParams := database.CreateChirpParams{
		Body:   cleanedBody,
		UserID: tokenUserID,
	}

	dbChirp, err := cfg.db.CreateChirp(r.Context(), chirpParams)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database error", err)
		return
	}

	response := chirpResponse{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserId:    dbChirp.UserID,
	}

	respondWithJSON(w, 201, response)
}

func replaceBadWords(s string) string {
	words := strings.Fields(s)
	for i, word := range words {
		lowercase := strings.ToLower(word)
		if lowercase == "kerfuffle" || lowercase == "sharbert" || lowercase == "fornax" {
			words[i] = "****"
		}
	}

	return strings.Join(words, " ")
}
