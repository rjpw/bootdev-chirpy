package postgres

import (
	"context"

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
	// TODO: generate UUID and timestamps, then call queries.CreateUser to insert into database
	return nil, nil
}

func (s *PostgresStore) GetUserByEmail(ctx context.Context, email string) (*store.User, error) {
	// TODO: implement GetUserByEmail
	return nil, nil
}

func (s *PostgresStore) GetUserByID(ctx context.Context, id string) (*store.User, error) {
	// TODO: implement GetUserByID
	return nil, nil
}

func (s *PostgresStore) DeleteUser(ctx context.Context, email string) error {
	// TODO: implement DeleteUser
	return nil
}

func (s *PostgresStore) UpdateUserEmail(ctx context.Context, oldEmail, newEmail string) error {
	// TODO: implement UpdateUserEmail
	return nil
}

func (s *PostgresStore) DeleteAllUsers(ctx context.Context) error {
	return nil
}
