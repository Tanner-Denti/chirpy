package auth

// TODO!
// Create tests that match up against tests in course, think more deeply about what the validator needs
// compared to just what the example had...

import (
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
)

const JWT_SECRET = "123456789"
const JWT_SECRET_2 = "qwertyuio"

func TestMakeJWT(t *testing.T) {
	tests := map[string]struct {
		userID uuid.UUID
		tokenSecret string
		expiresIn time.Duration
		want interface{}
	}{
		"has three parts": {userID: uuid.New(), tokenSecret: JWT_SECRET, expiresIn: 1 * time.Hour, want: 3},
	}
	
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tokenString, err := MakeJWT(tc.userID, tc.tokenSecret, tc.expiresIn)
			if err != nil {
				t.Fatalf("Unexpected error in %v: %v", name, err)
			}
			got := len(strings.Split(tokenString, "."))
			diff := cmp.Diff(tc.want, got)
			if diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestValidateJWT(t *testing.T) {
	userID := uuid.New()
	validTokenString, _ := MakeJWT(userID, JWT_SECRET, time.Hour)
	
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