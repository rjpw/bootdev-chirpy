package memory

// This file implements an in-memory user store.
// It is not intended for production use.

import (
	"sync"

	"github.com/google/uuid"
	"github.com/rjpw/bootdev-chirpy/internal/store"
)

type Store struct {
	mu    sync.RWMutex
	users map[uuid.UUID]store.User
}

func NewMemoryStore() *Store {
	return &Store{
		users: make(map[uuid.UUID]store.User),
	}
}
