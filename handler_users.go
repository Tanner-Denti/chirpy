package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Tanner-Denti/chirpy/internal/auth"
	"github.com/Tanner-Denti/chirpy/internal/database"
	"github.com/google/uuid"
)

type loginRequest struct {
	Email string `json:"email"`
	Password string `json:"password"`
}

type createUserRequest struct {
	Email string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email string `json:"email"`
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
	
	response := userResponse{
		ID: dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email: dbUser.Email,
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
		Email: createUserReq.Email,
		HashedPassword: hashedPassword,
	}

	dbUser, err := cfg.db.CreateUser(r.Context(), userParams)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Duplicates not allowed", err)
		return
	}

	response := userResponse{
		ID: dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email: dbUser.Email,
	}
	
	respondWithJSON(w, http.StatusCreated, response)
}