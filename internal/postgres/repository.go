package postgres

import (
	"database/sql"
	"errors"

	"github.com/lib/pq"
	"github.com/rjpw/bootdev-chirpy/internal/domain"
	"github.com/rjpw/bootdev-chirpy/internal/postgres/database"
)

type Repository struct {
	db *database.Queries
}

func NewPostgresRepository(db *database.Queries) *Repository {
	return &Repository{db: db}
}

// see https://www.postgresql.org/docs/current/errcodes-appendix.html
// for a list of Postgres error codes
func mapError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}
	var pgErr *pq.Error
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return domain.ErrConflict
		}
	}
	return err
}
