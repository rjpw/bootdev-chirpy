package testdb

import (
	"context"
	"database/sql"
	"io/fs"
	"sync"
	"testing"

	_ "github.com/lib/pq" // Postgres driver
	"github.com/pressly/goose/v3"
	"github.com/rjpw/bootdev-chirpy/internal/postgres/schema"
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

// SetupTestsuiteDB is for use in TestMain where *testing.T is not available.
// Returns the connection string, a cleanup function, and any error.
func SetupTestsuiteDB() (string, func(), error) {
	once.Do(setupCore)
	if setupErr != nil {
		return "", nil, setupErr
	}
	cleanup := func() {
		testDB.Close()
		postgresContainer.Terminate(context.Background())
	}
	return connStr, cleanup, nil
}

// SetupTestHelperDB is for use in individual tests. It calls setupCore via sync.Once
// and returns the EphemeralDB for direct DB access.
func SetupTestHelperDB(t *testing.T) EphemeralDB {
	t.Helper()
	once.Do(setupCore)
	if setupErr != nil {
		t.Fatalf("Failed to set up test database: %v", setupErr)
	}
	return EphemeralDB{
		DB:        testDB,
		Container: postgresContainer,
	}
}

func setupCore() {
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

	connStr, err = postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		setupErr = err
		return
	}

	testDB, err = sql.Open("postgres", connStr)
	if err != nil {
		setupErr = err
		return
	}

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

	if _, err = provider.Up(ctx); err != nil {
		setupErr = err
		return
	}

	testDB.Close()

	if err := postgresContainer.Snapshot(ctx); err != nil {
		setupErr = err
		return
	}

	testDB, err = sql.Open("postgres", connStr)
	if err != nil {
		setupErr = err
		return
	}
}
