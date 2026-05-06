package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
)

type Validatable interface {
	Validate() error
}

type contextKey string

const bodyKey contextKey = "validatedBody"

func withValidBody[T Validatable](next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body T
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			respondWithJSON(w, http.StatusBadRequest, jsonError{Error: err.Error()})
			return
		}
		if err := body.Validate(); err != nil {
			respondWithJSON(w, http.StatusBadRequest, jsonError{Error: err.Error()})
			return
		}
		ctx := context.WithValue(r.Context(), bodyKey, &body)
		next(w, r.WithContext(ctx))
	}
}

func validBody[T any](r *http.Request) *T {
	return r.Context().Value(bodyKey).(*T)
}
