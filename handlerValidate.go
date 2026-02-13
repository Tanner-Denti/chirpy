package main

import (
	"net/http"
	"encoding/json"
	"io"
	"strings"
)

func handlerValidateChirpPost(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	type requestBody struct {
		Body string `json:"body"`
	}
	type responseBody struct {
		CleanedBody string `json:"cleaned_body"`
	}

	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not read request", err) // How should I handle this error if it returns one?
		return
	}
	
	var req requestBody
	err = json.Unmarshal(reqBody, &req)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not unmarshal parameters", err) // How should I handle this error if it returns one?
		return
	}

	if len(req.Body) > 140 {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long", nil)
		return
	}

	cleanedBody := replaceBadWords(req.Body)

	respondWithJSON(w, 200, responseBody{CleanedBody: cleanedBody})
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