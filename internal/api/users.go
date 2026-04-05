package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rjpw/bootdev-chirpy/internal/store"
)

func (s *Server) CreateUser(ctx context.Context, email string) (*store.User, error) {
	return s.cfg.Users.CreateUser(ctx, email)
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	// email is a field in JSON body, so we need to parse the JSON body to get the email
	var payload struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	email := payload.Email
	user, err := s.CreateUser(r.Context(), email)
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			s.respondWithMessage(w, http.StatusConflict, "User already exists")
		} else {
			s.respondWithMessage(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	s.respondWithJSON(w, http.StatusCreated, user)
}
