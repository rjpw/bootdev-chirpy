package postgres

import (
	"database/sql"

	"github.com/rjpw/bootdev-chirpy/internal/application"
	"github.com/rjpw/bootdev-chirpy/internal/postgres/database"
)

func Open(url string) (*sql.DB, error) {
	// note: pg driver is imported by main in cmd/chirpy/main.go
	// and testdb in internal/testdb/testdb.go, so we don't need to import it here
	return sql.Open("postgres", url)
}

func NewPostgresRepositoryFromURL(dbURL string) (*application.Repositories, *sql.DB, error) {
	db, err := Open(dbURL)
	if err != nil {
		return nil, nil, err
	}
	return NewPostgresRepository(database.New(db)), db, nil
}
