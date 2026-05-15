package memory

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/rjpw/bootdev-chirpy/internal/application"
	"github.com/rjpw/bootdev-chirpy/internal/auth"
	"github.com/rjpw/bootdev-chirpy/internal/domain"
)

var _ application.UserRepository = (*Repository)(
	nil,
) // ensure MemoryStore implements the UserStore interface

func (repo *Repository) CreateUser(
	_ context.Context,
	email, password string,
) (*domain.User, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	// check for existing user with the same email
	for _, user := range repo.users {
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
	userCreds := domain.UserCredentials{
		ID:       id,
		Password: password,
	}

	repo.users[id] = user
	repo.userCredentials[id] = userCreds
	return &user, nil
}

func (repo *Repository) UpgradeUser(_ context.Context, id string) (*domain.User, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()

	parsedID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	user, ok := repo.users[parsedID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	user.IsChirpyRedMember = true

	repo.users[parsedID] = user
	return &user, nil
}

func (repo *Repository) GetUserByEmail(_ context.Context, email string) (*domain.User, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()

	for _, user := range repo.users {
		if user.Email == email {
			return &user, nil
		}
	}

	return nil, domain.ErrNotFound
}

func (repo *Repository) AuthenticateUser(
	ctx context.Context,
	email, password string,
) (*domain.User, error) {
	user, err := repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	creds := repo.userCredentials[user.ID]
	ok, err := auth.CheckPasswordHash(password, creds.Password)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("not authorized")
	}

	return user, nil
}

func (repo *Repository) GetUserByID(_ context.Context, id string) (*domain.User, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()

	parsedID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	user, ok := repo.users[parsedID]
	if !ok {
		return nil, domain.ErrNotFound
	}

	return &user, nil
}

func (repo *Repository) DeleteUser(_ context.Context, email string) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	for id, user := range repo.users {
		if user.Email == email {
			delete(repo.users, id)
			return nil
		}
	}

	return domain.ErrNotFound
}

func (repo *Repository) UpdateUserEmail(_ context.Context, oldEmail, newEmail string) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	var user *domain.User
	for id, u := range repo.users {
		if u.Email == oldEmail {
			user = &u
			delete(repo.users, id)
			break
		}
	}

	if user == nil {
		return domain.ErrNotFound
	}

	user.Email = newEmail
	user.UpdatedAt = time.Now().UTC().Truncate(time.Microsecond)
	repo.users[user.ID] = *user

	return nil
}

func (repo *Repository) DeleteAllUsers(_ context.Context) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	repo.users = make(map[uuid.UUID]domain.User)
	return nil
}

// unexported, no locking — caller must hold the lock
func (repo *Repository) getUserByID(id uuid.UUID) (*domain.User, error) {
	user, ok := repo.users[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &user, nil
}
