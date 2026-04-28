package memory

// This file implements an in-memory user store.
// It is not intended for production use.

import (
	"sync"

	"github.com/google/uuid"
	"github.com/rjpw/bootdev-chirpy/domain"
)

type Repository struct {
	mu    sync.RWMutex
	users map[uuid.UUID]domain.User
}

func NewMemoryRepository() *Repository {
	return &Repository{
		users: make(map[uuid.UUID]domain.User),
	}
}
