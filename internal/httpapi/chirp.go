package httpapi

import (
	"errors"
	"fmt"
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

	respondWithJSON(w, http.StatusCreated, chirp)
}

func (s *Server) handleGetChirps(w http.ResponseWriter, r *http.Request) {
	chirps, err := s.Repositories.Chirps.GetUserChirps(r.Context(), "%")
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		if errors.Is(err, domain.ErrNotFound) {
			respondWithMessage(w, http.StatusNotFound, err.Error())
		} else {
			respondWithMessage(w, http.StatusBadRequest, err.Error())
		}
	}

	respondWithJSON(w, http.StatusOK, chirps)

}

func (s *Server) handleGetChirp(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("chirpID")
	chirp, err := s.Repositories.Chirps.GetChirpByID(r.Context(), id)

	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		if errors.Is(err, domain.ErrNotFound) {
			respondWithMessage(w, http.StatusNotFound, err.Error())
		} else {
			respondWithMessage(w, http.StatusBadRequest, err.Error())
		}
	}

	respondWithJSON(w, http.StatusOK, chirp)

}
