package store

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

func (u User) String() string {
	return u.Email
}

func (u User) ShortID() string {
	return u.ID.String()[:8]
}

// marshal to JSON with short ID
func (u User) MarshalJSON() ([]byte, error) {
	type Alias User
	return []byte(`{"id":"` + u.ShortID() + `","created_at":"` + u.CreatedAt.Format(time.RFC3339) + `","updated_at":"` + u.UpdatedAt.Format(time.RFC3339) + `","email":"` + u.Email + `"}`), nil
}

type UserStore interface {
	CreateUser(ctx context.Context, email string) (User, error)
	GetUserByEmail(ctx context.Context, email string) (User, error)
	DeleteUser(ctx context.Context, email string) error
	UpdateUserEmail(ctx context.Context, oldEmail, newEmail string) (User, error)
	DeleteAllUsers(ctx context.Context) error
}
