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

// see https://www.postgresql.org/docs/current/errcodes-appendix.html
// for a list of Postgres error codes
func mapError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return store.ErrNotFound
	}
	var pgErr *pq.Error
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return store.ErrConflict
		}
	}
	return err
}
