package application

import (
	"context"
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/rjpw/bootdev-chirpy/internal/domain"
)

type Runnable interface {
	Run(ctx context.Context) error
	Close() error
}

type Environment struct {
	DBName    string
	DBURL     string
	Platform  string
	SecretKey string
}

type Repositories struct {
	Users        UserRepository
	UserSessions UserSessionRepository
	Chirps       ChirpRepository
}

type ChirpRepository interface {
	CreateChirp(ctx context.Context, body string, user_id uuid.UUID) (*domain.Chirp, error)
	GetChirpByID(ctx context.Context, id string) (*domain.Chirp, error)
	GetUserChirps(ctx context.Context, user_id string) ([]domain.Chirp, error)
	DeleteChirp(ctx context.Context, id string) error
	DeleteAllChirps(ctx context.Context, user_id string) error
}

type UserRepository interface {
	CreateUser(ctx context.Context, email, password string) (*domain.User, error)
	UpgradeUser(ctx context.Context, id string) (*domain.User, error)
	AuthenticateUser(ctx context.Context, email, password string) (*domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	GetUserByID(ctx context.Context, id string) (*domain.User, error)
	UpdateUserEmail(ctx context.Context, oldEmail, newEmail string) error
	DeleteUser(ctx context.Context, email string) error
	DeleteAllUsers(ctx context.Context) error
}

type UserSessionRepository interface {
	CreateSession(ctx context.Context, user_id uuid.UUID) (*domain.UserSession, error)
	GetSession(ctx context.Context, id string) (*domain.UserSession, error)
	RevokeSession(ctx context.Context, id string) error
	DeleteSessionsByUserID(ctx context.Context, user_id uuid.UUID) error
}

func LoadEnvironment() Environment {
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found: %v", err)
	}
	return Environment{
		DBName:    os.Getenv("DBNAME"),
		DBURL:     os.Getenv("DB_URL"),
		Platform:  os.Getenv("PLATFORM"),
		SecretKey: os.Getenv("HS256_KEY"),
	}
}
