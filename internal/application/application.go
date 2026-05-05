package application

import (
	"context"

	"github.com/rjpw/bootdev-chirpy/internal/domain"
)

type Runnable interface {
	Run(ctx context.Context) error
	Close() error
}

type Environment struct {
	DBName   string
	DBURL    string
	Platform string
}

type Repositories struct {
	Users  UserRepository
	Chirps ChirpRepository
}

type ChirpRepository interface {
	CreateChirp(ctx context.Context, body string) (*domain.Chirp, error)
	GetChirpByID(ctx context.Context, id string) (*domain.Chirp, error)
	DeleteChirp(ctx context.Context, id string) error
	DeleteAllChirps(ctx context.Context, user_id string) error
}

type UserRepository interface {
	CreateUser(ctx context.Context, email string) (*domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	GetUserByID(ctx context.Context, id string) (*domain.User, error)
	UpdateUserEmail(ctx context.Context, oldEmail, newEmail string) error
	DeleteUser(ctx context.Context, email string) error
	DeleteAllUsers(ctx context.Context) error
}
