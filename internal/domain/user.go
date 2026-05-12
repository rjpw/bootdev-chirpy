package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Email       string    `json:"email"`
	AccessToken string    `json:"token"`
	SessionID   string    `json:"refresh_token"`
}

type UserCredentials struct {
	ID       uuid.UUID
	Password string
}

type UserSession struct {
	ID        string
	UserID    uuid.UUID // allow cascading deletes and multiple devices
	CreatedAt time.Time
	UpdatedAt time.Time
	ExpiresAt time.Time
	RevokedAt time.Time // if note revoked, t.IsZero() will be true
}
