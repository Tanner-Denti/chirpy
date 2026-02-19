package main

import (
	"encoding/json"
	"net/http"

	"github.com/Tanner-Denti/chirpy/internal/auth"
	"github.com/google/uuid"
)

type polkaWebhook struct {
	Event string `json:"event"`
	Data struct {
		UserID string `json:"user_id"`
	} `json:"data"`
}

func (cfg *apiConfig) handlerPolkaWebhook(w http.ResponseWriter, r *http.Request) {
	reqApiKey, err := auth.GetAPIKey(r.Header)
	if err != nil || reqApiKey != cfg.polkaKey {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", err)
		return
	}
	
	defer r.Body.Close()
	var request polkaWebhook
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&request)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Bad request", err)
		return
	}

	if request.Event != "user.upgraded" {
		respondWithJSON(w, http.StatusNoContent, struct{}{})
		return
	}

	userID, err := uuid.Parse(request.Data.UserID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Bad request", err)
		return
	}

	_, err = cfg.db.UpdateUserChirpyRedByID(r.Context(), userID) 
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Bad request", err)
		return
	}
	
	respondWithJSON(w, http.StatusNoContent, struct{}{})
}