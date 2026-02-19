package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Tanner-Denti/chirpy/internal/auth"
	"github.com/Tanner-Denti/chirpy/internal/database"
	"github.com/google/uuid"
)

type loginRequest struct {
	Email            string `json:"email"`
	Password         string `json:"password"`
}

type createUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type updateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type jwtResponse struct {
	Token string `json:"token"`
}

type loginResponse struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	Token     string    `json:"token"`
	RefreshToken string `json:"refresh_token"`
	IsChirpyRed bool `json:"is_chirpy_red"`
}

type userResponse struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	IsChirpyRed bool `json:"is_chirpy_red"`
}

func (cfg *apiConfig) handleUpdateUser(w http.ResponseWriter, r * http.Request) {
	jwtToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized, no refresh token provided or invalid format, expected: Bearer <token>", err)
		return
	}

	userID, err := auth.ValidateJWT(jwtToken, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", err)
		return
	}

	defer r.Body.Close()
	var request updateUserRequest
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&request)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Bad request", err)
		return
	}

	hashedPassword, err := auth.HashPassword(request.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error", err)
		return
	}

	updateUserParams := database.UpdateUserByIDParams{
		Email: request.Email,
		HashedPassword: hashedPassword,
		ID: userID,
	}

	dbUser, err := cfg.db.UpdateUserByID(r.Context(), updateUserParams)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error", err)
		return
	}

	response := userResponse {
		ID: dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email: dbUser.Email,
		IsChirpyRed: dbUser.IsChirpyRed,
	}
	
	respondWithJSON(w, http.StatusOK, response)
}

func (cfg *apiConfig) handleRevoke(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "No refresh token provided or invalid format, expected: Bearer <token>", err)
		return
	}

	// If a token is already revoked, we should not allow the client to tamper with that time
	dbRefreshToken, _ := cfg.db.GetRefreshToken(r.Context(), refreshToken)
	if dbRefreshToken.RevokedAt.Valid {
		respondWithJSON(w, http.StatusNoContent, struct{}{})
		return
	}

	revokeRefreshTokenParams := database.RevokeRefreshTokenParams{
		RevokedAt: sql.NullTime{Time: time.Now(), Valid: true},
		Token: refreshToken,
	}

	err = cfg.db.RevokeRefreshToken(r.Context(), revokeRefreshTokenParams)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error", err)
		return
	}

	respondWithJSON(w, http.StatusNoContent, struct{}{})
}

func (cfg *apiConfig) handleRefresh(w http.ResponseWriter, r *http.Request) {

	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "No refresh token provided or invalid format, expected: Bearer <token>", err)
		return
	}
	
	// No refresh token matches, expired, or revoked
	dbRefreshToken, err := cfg.db.GetRefreshToken(r.Context(), refreshToken)
	if err != nil || dbRefreshToken.ExpiresAt.Before(time.Now()) || dbRefreshToken.RevokedAt.Valid {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", err)
		return
	}
	
	jwtToken, err := auth.MakeJWT(dbRefreshToken.UserID, cfg.jwtSecret, auth.DEFAULT_JWT_EXPIRATION)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Cannot make JWT token", err)
		return
	}

	respondWithJSON(w, http.StatusOK, jwtResponse{Token: jwtToken})
}

func (cfg *apiConfig) handleUserLogin(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var loginReq loginRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&loginReq)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Cannot decode request", err)
		return
	}

	dbUser, err := cfg.db.GetUserByEmail(r.Context(), loginReq.Email)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Incorrect email or password", err)
		return
	}

	match, err := auth.CheckPasswordHash(loginReq.Password, dbUser.HashedPassword)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Incorrect email or password", err)
		return
	} else if !match {
		respondWithError(w, http.StatusUnauthorized, "Incorrect email or password", nil)
		return
	}

	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error", err)
		return
	}

	refreshTokenParams := database.CreateRefreshTokenParams{
		Token: refreshToken,
		UserID: dbUser.ID,
		ExpiresAt: time.Now().Add(auth.DEFAULT_REFRESH_EXPIRATION),
	}

	dbRefreshToken, err := cfg.db.CreateRefreshToken(r.Context(), refreshTokenParams)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error: refresh token already exists", err)
		return
	}

	jwtToken, err := auth.MakeJWT(dbUser.ID, cfg.jwtSecret, auth.DEFAULT_JWT_EXPIRATION)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to create token", err)
		return
	}

	response := loginResponse{
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email:     dbUser.Email,
		Token:     jwtToken,
		RefreshToken: dbRefreshToken.Token,
		IsChirpyRed: dbUser.IsChirpyRed,
	}

	respondWithJSON(w, http.StatusOK, response)
}

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var createUserReq createUserRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&createUserReq)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Cannot decode request", err)
		return
	}

	hashedPassword, err := auth.HashPassword(createUserReq.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not hash password", err)
		return
	}

	userParams := database.CreateUserParams{
		Email:          createUserReq.Email,
		HashedPassword: hashedPassword,
	}

	dbUser, err := cfg.db.CreateUser(r.Context(), userParams)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Duplicates not allowed", err)
		return
	}

	response := userResponse{
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email:     dbUser.Email,
		IsChirpyRed: false,
	}

	respondWithJSON(w, http.StatusCreated, response)
}
