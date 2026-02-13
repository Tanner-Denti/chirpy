package main

import (
	"github.com/google/uuid"
	"net/http"
	"time"
	"encoding/json"
)

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	type requestBody struct {
		Email string `json:"email"`
	}
	type responseBody struct {
		ID uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email string `json:"email"`
	}
	
	var reqB requestBody
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&reqB)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Cannot decode request", err)
		return
	}

	user, err := cfg.db.CreateUser(r.Context(), reqB.Email)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Duplicates not allowed", err)
		return
	}

	resB := responseBody{
		ID: user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email: user.Email,
	}
	
	respondWithJSON(w, http.StatusCreated, resB)
}