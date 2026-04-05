package api

import (
	"encoding/json"
	"net/http"
	"strings"
)

// struct to receive a JSON api `chirp`
type parameters struct {
	Body string `json:"body"`
}

type jsonSuccess struct {
	CleanedBody string `json:"cleaned_body,omitempty"`
	Valid       bool   `json:"valid"`
}

func (p *parameters) Validate() bool {
	bodyLen := len(p.Body)
	return bodyLen > 0 && bodyLen <= 140
}

func (s *Server) handleValidateChirp(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	decoder := json.NewDecoder(r.Body)
	var decoded parameters
	if err := decoder.Decode(&decoded); err != nil {
		s.respondWithJSON(w, http.StatusBadRequest, jsonError{Error: err.Error()})
		return
	}

	if !decoded.Validate() {
		s.respondWithJSON(
			w,
			http.StatusBadRequest,
			jsonError{Error: "chirp body must be between 1 and 140 characters"},
		)
		return
	}

	s.respondWithJSON(
		w,
		http.StatusOK,
		jsonSuccess{CleanedBody: filterChirp(decoded.Body), Valid: true},
	)
}

func filterChirp(body string) string {
	badwords := []string{"sharbert", "kerfuffle", "fornax"}

	lowerBody := strings.ToLower(body)
	upperWords := strings.Split(body, " ")
	words := strings.Split(lowerBody, " ")

	for _, badword := range badwords {
		for i, word := range words {
			if word == badword {
				upperWords[i] = "****"
			}
		}
	}

	return strings.Join(upperWords, " ")
}
