package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

func (u User) ShortID() string {
	return u.ID.String()[:8]
}

type UserRepository interface {
	CreateUser(ctx context.Context, email string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	UpdateUserEmail(ctx context.Context, oldEmail, newEmail string) error
	DeleteUser(ctx context.Context, email string) error
	DeleteAllUsers(ctx context.Context) error
}
