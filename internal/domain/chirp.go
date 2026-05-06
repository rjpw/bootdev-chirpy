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
