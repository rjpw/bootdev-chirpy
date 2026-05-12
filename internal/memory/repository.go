package memory

// This file implements an in-memory user store.
// It is not intended for production use.

import (
	"sync"

	"github.com/google/uuid"
	"github.com/rjpw/bootdev-chirpy/internal/domain"
)

type Repository struct {
	mu              sync.RWMutex
	users           map[uuid.UUID]domain.User
	userCredentials map[uuid.UUID]domain.UserCredentials
	userSessions    map[string]domain.UserSession
	chirps          map[uuid.UUID]domain.Chirp
}

func NewMemoryRepository() *Repository {
	return &Repository{
		users:           make(map[uuid.UUID]domain.User),
		userCredentials: make(map[uuid.UUID]domain.UserCredentials),
		userSessions:    make(map[string]domain.UserSession),
		chirps:          make(map[uuid.UUID]domain.Chirp),
	}
}
