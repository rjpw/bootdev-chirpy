package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// struct to receive a JSON api `chirp`
type parameters struct {
	Body string `json:"body"`
}

type jsonError struct {
	Error string `json:"error"`
}

type jsonSuccess struct {
	CleanedBody string `json:"cleaned_body,omitempty"`
	Valid       bool   `json:"valid"`
}

func (p *parameters) Validate() bool {
	bodyLen := len(p.Body)
	return bodyLen > 0 && bodyLen <= 140
}

func (s *Server) respondWithJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	data, err := json.Marshal(payload)
	if err != nil {
		s.respondWithMessage(w, 500, "unexpected server error")
		return
	}
	fmt.Fprintf(w, "%s", string(data))
}

func (s *Server) respondWithMessage(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(statusCode)
	fmt.Fprintf(w, "%s", message)
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
		s.respondWithJSON(w, http.StatusBadRequest, jsonError{Error: "chirp body must be between 1 and 140 characters"})
		return
	}

	s.respondWithJSON(w, http.StatusOK, jsonSuccess{CleanedBody: filterChirp(decoded.Body), Valid: true})
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
