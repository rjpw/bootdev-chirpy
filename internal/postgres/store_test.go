//go:build integration

package postgres_test

import (
	"context"
	"testing"

	"github.com/rjpw/bootdev-chirpy/internal/postgres"
	"github.com/rjpw/bootdev-chirpy/internal/postgres/database"
	"github.com/rjpw/bootdev-chirpy/internal/postgres/testdb"
)

func setupTestRepository(t *testing.T) *postgres.Repository {
	t.Helper()
	ephemeralDB := testdb.Setup(t)
	t.Cleanup(func() {
		// Restore the database to its initial state after each test.
		if err := ephemeralDB.Container.Restore(context.Background()); err != nil {
			t.Fatalf("failed to restore test database container: %v", err)
		}
	})
	return postgres.NewPostgresRepository(database.New(ephemeralDB.DB))
}
