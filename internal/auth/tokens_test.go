package auth

import (
	"strings"
	"testing"
	"time"
	"net/http"

	"github.com/google/uuid"
)

const JWT_SECRET = "123456789"
const JWT_SECRET_2 = "qwertyuio"

func TestMakeJWT(t *testing.T) {
	tests := map[string]struct {
		userID uuid.UUID
		tokenSecret string
		expiresIn time.Duration
		wantPartsInString int
		wantErr bool
	}{
		"Valid inputs": {userID: uuid.New(), tokenSecret: JWT_SECRET, expiresIn: 1 * time.Hour, wantPartsInString: 3, wantErr: false},
		"Negative duration":  {userID: uuid.New(), tokenSecret: JWT_SECRET, expiresIn: -1 * time.Hour, wantPartsInString: 3, wantErr: false},
	}
	
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tokenString, err := MakeJWT(tc.userID, tc.tokenSecret, tc.expiresIn)
			if (err != nil) != tc.wantErr {
				t.Errorf("MakeJWT() error = %v - wanted error: %v", err, tc.wantErr)
			}
			
			got := len(strings.Split(tokenString, "."))
			if tc.wantPartsInString != got {
				t.Errorf("MakeJWT() got parts in string = %v - wanted: %v", got, tc.wantPartsInString)
			}
		})
	}
}

func TestValidateJWT(t *testing.T) {
	userID := uuid.New()
	validTokenString, _ := MakeJWT(userID, JWT_SECRET, time.Hour)
	expiredTokenString, _ := MakeJWT(userID, JWT_SECRET, -1 * time.Second)
	
	tests := map[string]struct {
		tokenString string
		tokenSecret string
		wantUserID uuid.UUID
		wantErr bool
	}{
		"Valid token": {
			tokenString: validTokenString,
			tokenSecret: JWT_SECRET, 
			wantUserID: userID,
			wantErr: false,
		},
		"Invalid token": {
			tokenString: "invalid.token.string",
			tokenSecret: JWT_SECRET, 
			wantUserID: uuid.UUID{},
			wantErr: true,
		},
		"Wrong secret": {
			tokenString: validTokenString,
			tokenSecret: JWT_SECRET_2,
			wantUserID: uuid.UUID{},
			wantErr: true,
		},
		"Expired token": {
			tokenString: expiredTokenString,
			tokenSecret: JWT_SECRET,
			wantUserID: uuid.UUID{},
			wantErr: true,
		},
	}
	
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			gotUserID, err := ValidateJWT(tc.tokenString, tc.tokenSecret)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateJWT() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if gotUserID != tc.wantUserID {
				t.Errorf("ValidateJWT() gotUserID = %v, want %v", gotUserID, tc.wantUserID)
			}
		})
	}
}

func TestGetBearerToken(t *testing.T) {
	const BEARER_TOKEN = "12345.qwerty.67890"
	const AUTHORIZATION_KEY = "Authorization"

	tests := map[string]struct {
		headerKey string
		headerVal string
		wantToken string
		wantErr bool
	}{
		"Valid authorization header": {
			headerKey: AUTHORIZATION_KEY,
			headerVal: "Bearer " + BEARER_TOKEN,
			wantToken: BEARER_TOKEN,
			wantErr: false,
		},
		"No Authorization header": {
			headerKey: "",
			headerVal: "Bearer " + BEARER_TOKEN,
			wantToken: "",
			wantErr: true,
		},
		"Invalid authorization field 1": {
			headerKey: AUTHORIZATION_KEY,
			headerVal: "Bearer" + BEARER_TOKEN,
			wantToken: "",
			wantErr: true,
		},
		"Invalid authorization field 2": {
			headerKey: AUTHORIZATION_KEY,
			headerVal: BEARER_TOKEN,
			wantToken: "",
			wantErr: true,
		},
		"Invalid authorization field 3": {
			headerKey: AUTHORIZATION_KEY,
			headerVal: "",
			wantToken: "",
			wantErr: true,
		},
		"Invalid authorization field 4": {
			headerKey: AUTHORIZATION_KEY,
			headerVal: "basic " + BEARER_TOKEN,
			wantToken: "",
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func (t *testing.T)  {
			headers := http.Header{}
			headers.Set(tc.headerKey, tc.headerVal)
			gotToken, err := GetBearerToken(headers)
			if (err != nil) != tc.wantErr {
				t.Errorf("GetBearerToken() got error: %v wanted error: %v", err, tc.wantErr)
			}
			if gotToken != tc.wantToken {
				t.Errorf("GetBearerToken() got token: %v wanted token: %v", gotToken, tc.wantToken)
			}
		})
	}
}