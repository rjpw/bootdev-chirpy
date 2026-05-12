package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/rjpw/bootdev-chirpy/internal/auth"
	"github.com/rjpw/bootdev-chirpy/internal/domain"
)

// struct to receive a JSON api `chirp`
type PostLoginRequest struct {
	Email            string `json:"email"`
	Password         string `json:"password"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var params PostLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Cannot decode User from request body", http.StatusBadRequest)
		return
	}
	email := params.Email
	hashedPassword, err := auth.HashPassword(params.Password)
	user, err := s.Repositories.Users.CreateUser(r.Context(), email, hashedPassword)
	if err != nil {
		if errors.Is(err, domain.ErrConflict) {
			respondWithMessage(w, http.StatusConflict, "User already exists")
		} else {
			respondWithMessage(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	respondWithJSON(w, http.StatusCreated, user)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var params PostLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Cannot decode User from request body", http.StatusBadRequest)
		return
	}

	user, err := s.Repositories.Users.AuthenticateUser(
		r.Context(), params.Email, params.Password)
	if err != nil {
		fmt.Printf("User authentication error: %v\n", err)
		if errors.Is(err, domain.ErrNotFound) {
			respondWithMessage(w, http.StatusNotFound, "User not found")
		} else {
			respondWithMessage(w, http.StatusUnauthorized, "Not authorized")
		}
		return
	}

	// choose the minimum of 3600 seconds and the user's requested expires_in_seconds
	minExpiry := 3600
	if params.ExpiresInSeconds > 0 && params.ExpiresInSeconds < minExpiry {
		minExpiry = params.ExpiresInSeconds
	}

	// generate a token and attach to the user object
	token, err := auth.MakeJWT(user.ID, s.environment.SecretKey, time.Duration(minExpiry)*time.Second)
	user.AccessToken = token

	respondWithJSON(w, http.StatusOK, user)
}
