package memory

// This file implements an in-memory user store.
// It is not intended for production use.

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rjpw/bootdev-chirpy/internal/store"
)

type MemoryStore struct {
	mu    sync.RWMutex
	users map[uuid.UUID]store.User
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		users: make(map[uuid.UUID]store.User),
	}
}

func (s *MemoryStore) CreateUser(ctx context.Context, email string) (store.User, error) {

	s.mu.Lock()
	defer s.mu.Unlock()

	// check for existing user with the same email
	for _, user := range s.users {
		if user.Email == email {
			return store.User{}, store.ErrConflict
		}
	}

	id := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)
	user := store.User{
		ID:        id,
		CreatedAt: now,
		UpdatedAt: now,
		Email:     email,
	}

	s.users[id] = user
	return user, nil
}
