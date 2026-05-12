package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

func FilterChirp(body string) string {
	// TODO: make this configurable
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

	return truncateChirp(strings.Join(upperWords, " "), 140)
}

// The method of truncation isn't terribly opinionated, but it does allow for UTF8 multi-byte runes.
func truncateChirp(s string, maxLength int) string {
	r := []rune(s)
	if len(r) > maxLength {
		return string(r[:maxLength])
	}
	return s
}
