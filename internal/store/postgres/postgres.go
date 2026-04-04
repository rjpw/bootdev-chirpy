package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rjpw/bootdev-chirpy/internal/database"
	"github.com/rjpw/bootdev-chirpy/internal/store"
)

type PostgresStore struct {
	db *database.Queries
}

func NewPostgresStore(db *database.Queries) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) CreateUser(ctx context.Context, email string) (*store.User, error) {
	user, err := s.db.CreateUser(ctx, database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Email:     email,
	})
	if err != nil {
		return nil, err
	}
	return &store.User{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt}, nil
}

func (s *PostgresStore) GetUserByEmail(ctx context.Context, email string) (*store.User, error) {
	user, err := s.db.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	return &store.User{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt}, nil
}

func (s *PostgresStore) GetUserByID(ctx context.Context, id string) (*store.User, error) {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	user, err := s.db.GetUserByID(ctx, parsedID)
	if err != nil {
		return nil, err
	}
	return &store.User{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt}, nil
}

func (s *PostgresStore) DeleteUser(ctx context.Context, email string) error {
	user, err := s.db.GetUserByEmail(ctx, email)
	if err != nil {
		return err
	}
	return s.db.DeleteUser(ctx, user.ID)
}

func (s *PostgresStore) UpdateUserEmail(ctx context.Context, oldEmail, newEmail string) error {
	_, err := s.db.GetUserByEmail(ctx, newEmail)
	if err == nil {
		return store.ErrConflict
	}
	_, err = s.db.GetUserByEmail(ctx, oldEmail)
	if err != nil {
		return err
	}
	_, err = s.db.UpdateUser(ctx, database.UpdateUserParams{
		Email:     oldEmail,
		UpdatedAt: time.Now(),
		Email_2:   newEmail,
	})
	return err
}

func (s *PostgresStore) DeleteAllUsers(ctx context.Context) error {
	return s.db.DeleteAllUsers(ctx)
}
