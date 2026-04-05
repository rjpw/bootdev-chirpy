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

type EphemeralDB struct {
	DB        *sql.DB
	Container *postgres.PostgresContainer
}

var (
	once              sync.Once
	setupErr          error
	connStr           string
	testDB            *sql.DB
	postgresContainer *postgres.PostgresContainer
)

func Setup(t *testing.T) EphemeralDB {
	t.Helper()
	once.Do(func() {
		ctx := context.Background()

		var err error
		postgresContainer, err = postgres.Run(ctx,
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
		connStr, err = postgresContainer.ConnectionString(ctx, "sslmode=disable")
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

		// define a subdirectory for the migrations to avoid including non-migration files in the root of the schema package
		migrationsFS, err := fs.Sub(schema.Migrations, "migrations")
		if err != nil {
			setupErr = err
			return
		}

		// Create a new goose provider using the embedded migrations.
		provider, err := goose.NewProvider(
			goose.DialectPostgres,
			testDB,
			migrationsFS,
		)
		if err != nil {
			setupErr = err
			return
		}

		// Apply all up migrations.
		results, err := provider.Up(ctx)
		if err != nil {
			setupErr = err
			return
		}
		t.Logf("Migrations applied: %v", results)

		// close the connection before snapshot
		testDB.Close()

		// Snapshot the container to speed up future test runs.
		if err := postgresContainer.Snapshot(ctx); err != nil {
			setupErr = err
			return
		}

		// reopen after snapshot
		testDB, err = sql.Open("postgres", connStr)
		if err != nil {
			setupErr = err
			return
		}
	})

	if setupErr != nil {
		t.Fatalf("Failed to set up test database: %v", setupErr)
	}

	return EphemeralDB{
		DB:        testDB,
		Container: postgresContainer,
	}
}
