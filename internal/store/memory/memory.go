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

func (s *MemoryStore) GetUserByEmail(ctx context.Context, email string) (store.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, user := range s.users {
		if user.Email == email {
			return user, nil
		}
	}

	return store.User{}, store.ErrNotFound
}

func (s *MemoryStore) DeleteUser(ctx context.Context, email string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, user := range s.users {
		if user.Email == email {
			delete(s.users, id)
			return nil
		}
	}

	return store.ErrNotFound
}

func (s *MemoryStore) UpdateUserEmail(ctx context.Context, oldEmail, newEmail string) (store.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var user *store.User
	for id, u := range s.users {
		if u.Email == oldEmail {
			user = &u
			delete(s.users, id)
			break
		}
	}

	if user == nil {
		return store.User{}, store.ErrNotFound
	}

	user.Email = newEmail
	user.UpdatedAt = time.Now().UTC().Truncate(time.Microsecond)
	s.users[user.ID] = *user

	return *user, nil
}

func (s *MemoryStore) DeleteAllUsers(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.users = make(map[uuid.UUID]store.User)
	return nil
}
