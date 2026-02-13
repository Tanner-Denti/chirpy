package main

import (
	"github.com/google/uuid"
	"github.com/Tanner-Denti/chirpy/internal/database"
	"net/http"
	"encoding/json"
	"io"
	"strings"
	"time"
)

type chirpResponse struct {
	ID uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body string `json:"body"`
	UserId uuid.UUID `json:"user_id"`
}

type chirpsResponse []chirpResponse

func (cfg *apiConfig) handlerGetChirpByID(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	chirpID := r.PathValue("chirpID")
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
		ID: dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body: dbChirp.Body,
		UserId: dbChirp.UserID,
	}

	respondWithJSON(w, http.StatusOK, response)
}

func (cfg *apiConfig) handlerGetAllChirps(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	dbChirps, err := cfg.db.GetAllChirps(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database error", err)
		return
	}

	chirps := chirpsResponse{}
	for _, dbChirp := range dbChirps {
		chirps = append(chirps, chirpResponse{
			ID: dbChirp.ID,
			CreatedAt: dbChirp.CreatedAt,
			UpdatedAt: dbChirp.UpdatedAt,
			Body: dbChirp.Body,
			UserId: dbChirp.UserID,
		})
	}

	respondWithJSON(w, http.StatusOK, chirps)
}

func (cfg *apiConfig) handlerCreateChirp(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	type requestBody struct {
		Body string `json:"body"`
		UserID uuid.UUID `json:"user_id"`
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
		Body: cleanedBody,
		UserID: req.UserID,
	}

	dbChirp, err := cfg.db.CreateChirp(r.Context(), chirpParams)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database error", err)
		return 
	}
	
	response := chirpResponse{
		ID: dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body: dbChirp.Body,
		UserId: dbChirp.UserID,
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