package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"sort"

	// "github.com/google/uuid"

	"github.com/rjpw/bootdev-chirpy/internal/auth"
	"github.com/rjpw/bootdev-chirpy/internal/domain"
)

// struct to receive a JSON api `chirp`
type ChirpParams struct {
	Body string `json:"body"`
	// UserID uuid.UUID `json:"user_id"`
}

func (p ChirpParams) Validate() error {
	bodyLen := len(p.Body)
	if bodyLen < 1 || bodyLen > 140 {
		return errors.New("Chirp body must be between 1 and 140 characters inclusive.")
	}
	return nil
}

func (router *ChirpyAPIRouter) handleCreateChirp(w http.ResponseWriter, r *http.Request) {
	params := validBody[ChirpParams](r)

	// TODO: see if we can rely on the middleware, and simply remove this as redundant
	cleaned := domain.FilterChirp(params.Body)

	// TODO: make this middleware
	// get the user ID from the bearer token
	token, err := auth.GetAccessToken(r.Header)
	if err != nil {
		respondWithMessage(w, http.StatusUnauthorized, err.Error())
	}
	userID, err := auth.ValidateJWT(token, router.environment.SecretKey)
	if err != nil {
		respondWithMessage(w, http.StatusUnauthorized, err.Error())
	}

	chirp, err := router.Repositories.Chirps.CreateChirp(r.Context(), cleaned, userID)
	if err != nil {
		if errors.Is(err, domain.ErrConflict) {
			respondWithMessage(w, http.StatusConflict, "Chirp already exists")
		} else {
			respondWithMessage(w, http.StatusBadRequest, err.Error())
		}
	}

	respondWithJSON(w, http.StatusCreated, chirp)
}

func (router *ChirpyAPIRouter) handleGetChirps(w http.ResponseWriter, r *http.Request) {
	paramAuthorID := r.URL.Query().Get("author_id")
	paramSortOrder := r.URL.Query().Get("sort")

	author_id := "%"
	sortOrder := "asc"

	if len(paramAuthorID) > 0 {
		author_id = paramAuthorID
	}

	if len(paramSortOrder) > 0 && paramSortOrder == "desc" {
		sortOrder = paramSortOrder
	}

	chirps, err := router.Repositories.Chirps.GetUserChirps(r.Context(), author_id)
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		if errors.Is(err, domain.ErrNotFound) {
			respondWithMessage(w, http.StatusNotFound, err.Error())
		} else {
			respondWithMessage(w, http.StatusBadRequest, err.Error())
		}
	}

	if sortOrder == "desc" {
		sort.Slice(chirps, func(i, j int) bool { return chirps[i].CreatedAt.After(chirps[j].CreatedAt) })
	}

	respondWithJSON(w, http.StatusOK, chirps)
}

func (router *ChirpyAPIRouter) handleGetChirp(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("chirpID")
	chirp, err := router.Repositories.Chirps.GetChirpByID(r.Context(), id)
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

func (router *ChirpyAPIRouter) handleDeleteChirp(w http.ResponseWriter, r *http.Request) {
	accessToken, err := auth.GetAccessToken(r.Header)
	if err != nil {
		http.Error(
			w,
			fmt.Sprintf("Error retrieving access token: %s", err.Error()),
			http.StatusUnauthorized,
		)
		return
	}

	user_id, err := auth.ValidateJWT(accessToken, router.environment.SecretKey)
	if err != nil {
		http.Error(
			w,
			fmt.Sprintf("Error validating access token: %s", err.Error()),
			http.StatusUnauthorized,
		)
		return
	}

	chirpID := r.PathValue("chirpID")
	chirp, err := router.Repositories.Chirps.GetChirpByID(r.Context(), chirpID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Chirp with ID %s not found", err.Error()), http.StatusNotFound)
		return
	}

	if chirp.UserID != user_id {
		http.Error(
			w,
			fmt.Sprintf(
				"Cannot access Chirp: %s owned by %s with UserID: %s",
				chirpID,
				chirp.UserID,
				user_id,
			),
			http.StatusForbidden,
		)
		return
	}

	err = router.Repositories.Chirps.DeleteChirp(r.Context(), chirpID)
	if err != nil {
		http.Error(
			w,
			fmt.Sprintf("Error deleting chirp: %s", err.Error()),
			http.StatusInternalServerError,
		)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
