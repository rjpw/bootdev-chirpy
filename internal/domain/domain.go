package domain

import "github.com/google/uuid"

func ShortID(id uuid.UUID) string {
	return id.String()[:8]
}
