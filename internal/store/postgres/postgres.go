package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rjpw/bootdev-chirpy/internal/database"
	"github.com/rjpw/bootdev-chirpy/internal/store"
)

type PostgresStore struct {
	db *database.Queries
}

var _ store.UserStore = (*PostgresStore)(nil) // ensure PostgresStore implements the UserStore interface

func NewPostgresStore(db *database.Queries) *PostgresStore {
	return &PostgresStore{db: db}
}

func mapError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return store.ErrNotFound
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return store.ErrConflict
	}
	return err
}

func (s *PostgresStore) CreateUser(ctx context.Context, email string) (*store.User, error) {
	now := time.Now().UTC().Truncate(time.Microsecond)
	user, err := s.db.CreateUser(ctx, database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		Email:     email,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &store.User{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt}, nil
}

func (s *PostgresStore) GetUserByEmail(ctx context.Context, email string) (*store.User, error) {
	user, err := s.db.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, mapError(err)
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
		return nil, mapError(err)
	}
	return &store.User{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt}, nil
}

func (s *PostgresStore) DeleteUser(ctx context.Context, email string) error {
	user, err := s.db.GetUserByEmail(ctx, email)
	if err != nil {
		return mapError(err)
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
		return mapError(err)
	}
	_, err = s.db.UpdateUser(ctx, database.UpdateUserParams{
		Email:     oldEmail,
		UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
		Email_2:   newEmail,
	})
	return mapError(err)
}

func (s *PostgresStore) DeleteAllUsers(ctx context.Context) error {
	return s.db.DeleteAllUsers(ctx)
}
