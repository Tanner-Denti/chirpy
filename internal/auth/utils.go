package auth

import (
	"net/http"
	"errors"
	"strings"
)

func GetBearerToken(headers http.Header) (string, error) {
	authField := headers.Get("Authorization")
	if authField == "" {
		return "", errors.New("no authorization header field found")
	}
	
	parts := strings.Fields(authField)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", errors.New("invalid authorization header field: expected 'Bearer header.claims.sig'")
	}

	bearerToken := parts[len(parts) - 1]

	return bearerToken, nil
}

func GetAPIKey(headers http.Header) (string, error) {
	authField := headers.Get("Authorization")
	if authField == "" {
		return "", errors.New("no authorization header field found")
	}
	
	parts := strings.Fields(authField)
	if len(parts) != 2 || parts[0] != "ApiKey" {
		return "", errors.New("invalid authorization header field: expected 'ApiKey THE_KEY_HERE'")
	}

	apiKey := parts[len(parts) - 1]

	return apiKey, nil
}