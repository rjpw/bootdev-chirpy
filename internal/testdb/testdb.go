package testdb

import (
	"context"
	"database/sql"
	"io/fs"
	"sync"
	"testing"

	_ "github.com/lib/pq" // Postgres driver
	"github.com/pressly/goose/v3"
	"github.com/rjpw/bootdev-chirpy/internal/schema"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var (
	once     sync.Once
	setupErr error
	testDB   *sql.DB
)

func Setup(t *testing.T) *sql.DB {
	t.Helper()
	once.Do(func() {
		ctx := context.Background()
		postgresContainer, err := postgres.Run(ctx,
			"postgres:17.2-alpine3.21",
			postgres.WithDatabase("test"),
			postgres.WithUsername("user"),
			postgres.WithPassword("password"),
			postgres.BasicWaitStrategies(),
		)
		if err != nil {
			setupErr = err
			return
		}

		// Get the connection string for the database.
		connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			setupErr = err
			return
		}

		// Open a connection to the database.
		testDB, err = sql.Open("postgres", connStr)
		if err != nil {
			setupErr = err
			return
		}

		// Ping the database to ensure it's up and running.
		if err := testDB.PingContext(ctx); err != nil {
			setupErr = err
			return
		}

		migrationsFS, err := fs.Sub(schema.Migrations, "migrations")
		if err != nil {
			setupErr = err
			return
		}

		provider, err := goose.NewProvider(
			goose.DialectPostgres,
			testDB,
			migrationsFS,
		)
		if err != nil {
			setupErr = err
			return
		}

		results, err := provider.Up(ctx)
		if err != nil {
			setupErr = err
			return
		}
		t.Logf("Migrations applied: %v", results)
	})

	if setupErr != nil {
		t.Fatalf("Failed to set up test database: %v", setupErr)
	}

	return testDB
}
