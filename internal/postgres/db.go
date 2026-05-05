package postgres

import (
	"database/sql"

	"github.com/rjpw/bootdev-chirpy/internal/postgres/database"
)

func Open(url string) (*sql.DB, error) {
	return sql.Open("postgres", url)
}

func NewRepositoryFromURL(dbURL string) (*Repository, *sql.DB, error) {
	db, err := Open(dbURL)
	if err != nil {
		return nil, nil, err
	}
	return NewPostgresRepository(database.New(db)), db, nil
}
