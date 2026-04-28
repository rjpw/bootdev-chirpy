//go:build integration

package testdb_test

import (
	"testing"

	"github.com/rjpw/bootdev-chirpy/internal/postgres/testdb"
)

func TestSetup(t *testing.T) {
	db := testdb.Setup(t)

	// Can we reach the database?
	if err := db.DB.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}

	// Did migrations run?
	var tableName string
	err := db.DB.QueryRow(
		"SELECT table_name FROM information_schema.tables WHERE table_name = $1",
		"users",
	).Scan(&tableName)
	if err != nil {
		t.Fatalf("users table not found: %v", err)
	}
}
