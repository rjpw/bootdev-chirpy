package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/rjpw/bootdev-chirpy/internal/auth"
	"github.com/rjpw/bootdev-chirpy/internal/domain"
)

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	// email is a field in JSON body, so we need to parse the JSON body to get the email
	var payload struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	email := payload.Email
	hashedPassword, err := auth.HashPassword(payload.Password)
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
	// email is a field in JSON body, so we need to parse the JSON body to get the email
	var payload struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	user, err := s.Repositories.Users.AuthenticateUser(
		r.Context(), payload.Email, payload.Password)
	if err != nil {
		fmt.Printf("User authentication error: %v\n", err)
		if errors.Is(err, domain.ErrNotFound) {
			respondWithMessage(w, http.StatusNotFound, "User not found")
		} else {
			respondWithMessage(w, http.StatusUnauthorized, "Not authorized")
		}
		return
	}

	respondWithJSON(w, http.StatusOK, user)
}
