package postgres

import (
	"database/sql"
	"errors"

	"github.com/lib/pq"
	"github.com/rjpw/bootdev-chirpy/internal/database"
	"github.com/rjpw/bootdev-chirpy/internal/store"
)

type Store struct {
	db *database.Queries
}

func NewPostgresStore(db *database.Queries) *Store {
	return &Store{db: db}
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
