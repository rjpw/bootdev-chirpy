package memory

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rjpw/bootdev-chirpy/domain"
)

var _ domain.UserRepository = (*Repository)(nil) // ensure MemoryStore implements the UserStore interface

func (s *Repository) CreateUser(_ context.Context, email string) (*domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// check for existing user with the same email
	for _, user := range s.users {
		if user.Email == email {
			return nil, domain.ErrConflict
		}
	}

	id := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)
	user := domain.User{
		ID:        id,
		CreatedAt: now,
		UpdatedAt: now,
		Email:     email,
	}

	s.users[id] = user
	return &user, nil
}

func (s *Repository) GetUserByEmail(_ context.Context, email string) (*domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, user := range s.users {
		if user.Email == email {
			return &user, nil
		}
	}

	return nil, domain.ErrNotFound
}

func (s *Repository) GetUserByID(_ context.Context, id string) (*domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	parsedID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	user, ok := s.users[parsedID]
	if !ok {
		return nil, domain.ErrNotFound
	}

	return &user, nil
}

func (s *Repository) DeleteUser(_ context.Context, email string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, user := range s.users {
		if user.Email == email {
			delete(s.users, id)
			return nil
		}
	}

	return domain.ErrNotFound
}

func (s *Repository) UpdateUserEmail(_ context.Context, oldEmail, newEmail string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var user *domain.User
	for id, u := range s.users {
		if u.Email == oldEmail {
			user = &u
			delete(s.users, id)
			break
		}
	}

	if user == nil {
		return domain.ErrNotFound
	}

	user.Email = newEmail
	user.UpdatedAt = time.Now().UTC().Truncate(time.Microsecond)
	s.users[user.ID] = *user

	return nil
}

func (s *Repository) DeleteAllUsers(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.users = make(map[uuid.UUID]domain.User)
	return nil
}
