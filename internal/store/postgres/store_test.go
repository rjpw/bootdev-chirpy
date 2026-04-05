package postgres_test

import (
	"context"
	"testing"

	"github.com/rjpw/bootdev-chirpy/internal/database"
	"github.com/rjpw/bootdev-chirpy/internal/store/postgres"
	"github.com/rjpw/bootdev-chirpy/internal/testdb"
)

func setupTestStore(t *testing.T) *postgres.Store {
	t.Helper()
	ephemeralDB := testdb.Setup(t)
	t.Cleanup(func() {
		// Restore the database to its initial state after each test.
		if err := ephemeralDB.Container.Restore(context.Background()); err != nil {
			t.Fatalf("failed to restore test database container: %v", err)
		}
	})
	return postgres.NewPostgresStore(database.New(ephemeralDB.DB))
}
