package auth_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rjpw/bootdev-chirpy/internal/application"
	"github.com/rjpw/bootdev-chirpy/internal/auth"
)

func TestPasswordHashing(t *testing.T) {
	cases := []struct {
		name         string
		password     string
		hash         string
		generateHash bool
		wantMatch    bool
	}{
		{
			name:      "should fail blank hash",
			password:  "password",
			hash:      "",
			wantMatch: false,
		},
		{
			name:         "check generated hash",
			password:     "password",
			generateHash: true,
			wantMatch:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.generateHash {
				hash, err := auth.HashPassword(tc.password)
				if err != nil {
					t.Errorf("Unexpected error %v", err)
				}
				tc.hash = hash
			}

			gotMatch, _ := auth.CheckPasswordHash(tc.password, tc.hash)
			if gotMatch != tc.wantMatch {
				t.Errorf("Expected %s to get %v and got %v", tc.name, tc.wantMatch, gotMatch)
			}
		})
	}
}

func TestJWTCreation(t *testing.T) {
	// note: this loads internal/auth/testdata/.env
	env := application.LoadEnvironment()
	cases := []struct {
		name            string
		user_id         string
		expiresIn       time.Duration
		sleepFor        time.Duration
		usesGoodSecret  bool
		creationError   error
		validationError error
	}{
		{
			name:            "valid and timely",
			user_id:         "bf1b298a-7e73-4aa1-b8d2-84baa7ef38ae",
			expiresIn:       1 * time.Second,
			sleepFor:        10 * time.Millisecond,
			usesGoodSecret:  true,
			creationError:   nil,
			validationError: nil,
		},
		{
			name:            "valid but late",
			user_id:         "bf1b298a-7e73-4aa1-b8d2-84baa7ef38ae",
			expiresIn:       100 * time.Millisecond,
			sleepFor:        200 * time.Millisecond,
			usesGoodSecret:  true,
			creationError:   nil,
			validationError: jwt.ErrTokenExpired,
		},
		{
			name:            "valid but used bad secret",
			user_id:         "bf1b298a-7e73-4aa1-b8d2-84baa7ef38ae",
			expiresIn:       1 * time.Second,
			sleepFor:        200 * time.Millisecond,
			usesGoodSecret:  false,
			creationError:   nil,
			validationError: jwt.ErrTokenSignatureInvalid,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id, err := uuid.Parse(tc.user_id)

			// default to the riskier state "imagine-hacker-trying-keys"
			secret := uuid.New().String()
			if tc.usesGoodSecret {
				secret = env.SecretKey
			}

			token, err := auth.MakeJWT(id, secret, tc.expiresIn)
			fmt.Printf("Token: %s\n\n", token)
			if err != tc.creationError {
				t.Errorf("Expecting creation error %v and got %v", tc.creationError, err)
			}

			time.Sleep(tc.sleepFor)

			// always validate with the correct token (as the runtime system will)
			actualID, err := auth.ValidateJWT(token, env.SecretKey)
			if tc.validationError != nil {

				fmt.Printf("Expecting validation error: %v\n", tc.validationError)

				if err == nil {
					t.Errorf("Expecting validation error %v and got none", tc.validationError)
				} else if !errors.Is(err, tc.validationError) {
					t.Errorf("Expecting validation error %v and got %v", tc.validationError, err)
				}

			} else if actualID != id {
				t.Errorf("Expecting UUID %v and got %v", tc.user_id, actualID)
			} else {
				fmt.Printf("Good news: %s matches %s", actualID, tc.user_id)
			}
		})
	}
}

func TestBearerTokens(t *testing.T) {
	cases := []struct {
		name          string
		headers       http.Header
		expectedError error
	}{
		{
			name: "good example",
			headers: map[string][]string{
				"Authorization": {
					"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJjaGlycHktYWNjZXNzIiwic3ViIjoiYmYxYjI5OGEtN2U3My00YWExLWI4ZDItODRiYWE3ZWYzOGFlIiwiZXhwIjoxNzc4MTY4NTY5LCJpYXQiOjE3NzgxNjg1Njh9.M9nKwDrqKHSye8jsUzVD2i7C2p4aebpWRCSmPxO8Yr8",
				},
				"Accept": {"application/json"},
			},
			expectedError: nil,
		},
		{
			name: "missing authorization",
			headers: map[string][]string{
				"Accept-Encoding": {"gzip, deflate"},
				"Accept-Language": {"en-us"},
				"Foo":             {"Bar", "two"},
			},
			expectedError: errors.New("No valid Bearer token found"),
		},
		{
			name: "missing bearer prefix",
			headers: map[string][]string{
				"Accept-Encoding": {"gzip, deflate"},
				"Accept-Language": {"en-us"},
				"Authorization": {
					"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJjaGlycHktYWNjZXNzIiwic3ViIjoiYmYxYjI5OGEtN2U3My00YWExLWI4ZDItODRiYWE3ZWYzOGFlIiwiZXhwIjoxNzc4MTY4NTY5LCJpYXQiOjE3NzgxNjg1Njh9.M9nKwDrqKHSye8jsUzVD2i7C2p4aebpWRCSmPxO8Yr8",
				},
			},
			expectedError: errors.New("No valid Bearer token found"),
		},
		{
			name: "malformed token",
			headers: map[string][]string{
				"Accept-Language": {"en-us"},
				"Authorization":   {"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"},
				"Accept-Encoding": {"gzip, deflate"},
			},
			expectedError: errors.New("No valid Bearer token found"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := auth.GetAccessToken(tc.headers)
			if tc.expectedError != nil {
				if err == nil {
					t.Errorf("Expecting error %v and got none", tc.expectedError)
				} else if err.Error() != tc.expectedError.Error() {
					t.Errorf("Expecting error %v and got %v", tc.expectedError, err)
				}
			}
		})
	}
}
