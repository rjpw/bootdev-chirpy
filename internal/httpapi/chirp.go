package httpapi

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/rjpw/bootdev-chirpy/internal/domain"
)

// struct to receive a JSON api `chirp`
type ChirpParams struct {
	Body   string    `json:"body"`
	UserID uuid.UUID `json:"user_id"`
}

func (p ChirpParams) Validate() error {
	bodyLen := len(p.Body)
	if bodyLen < 1 || bodyLen > 140 {
		return errors.New("Chirp body must be between 1 and 140 characters inclusive.")
	}
	return nil
}

func (s *Server) handleCreateChirp(w http.ResponseWriter, r *http.Request) {
	params := validBody[ChirpParams](r)
	cleaned := domain.FilterChirp(params.Body)

	chirp, err := s.Repositories.Chirps.CreateChirp(r.Context(), cleaned, params.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrConflict) {
			respondWithMessage(w, http.StatusConflict, "Chirp already exists")
		} else {
			respondWithMessage(w, http.StatusBadRequest, err.Error())
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	respondWithJSON(w, http.StatusCreated, chirp)
}
