package postgres

import (
	"database/sql"

	"github.com/rjpw/bootdev-chirpy/internal/database"
)

func Open(url string) (*sql.DB, error) {
	return sql.Open("postgres", url)
}

func NewPostgresStoreFromURL(dbURL string) (*Store, *sql.DB, error) {
	db, err := sql.Open("postgres", dbURL) // note: pg driver is imported in cmd/chirpy/main.go
	if err != nil {
		return nil, nil, err
	}
	return NewPostgresStore(database.New(db)), db, nil
}
