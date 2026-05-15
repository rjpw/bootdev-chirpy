package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID                uuid.UUID
	CreatedAt         time.Time
	UpdatedAt         time.Time
	Email             string
	IsChirpyRedMember bool
}

type UserCredentials struct {
	ID       uuid.UUID
	Password string
}

type UserSession struct {
	ID          string
	AccessToken string
	UserID      uuid.UUID // allow cascading deletes and multiple devices
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ExpiresAt   time.Time
	RevokedAt   time.Time // if note revoked, t.IsZero() will be true
}
