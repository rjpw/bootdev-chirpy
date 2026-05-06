//go:build integration

package testdb_test

import (
	"testing"

	"github.com/rjpw/bootdev-chirpy/internal/postgres/testdb"
)

func TestAllTablesExist(t *testing.T) {

	// struct is overkill in this case, but it might be useful later
	cases := []struct {
		name string
	}{
		{
			name: "users",
		},
		{
			name: "chirps",
		},
	}

	db := testdb.Setup(t)

	// Can we reach the database?
	if err := db.DB.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			var tableName string
			err := db.DB.QueryRow(
				"SELECT table_name FROM information_schema.tables WHERE table_name = $1",
				tc.name,
			).Scan(&tableName)
			if err != nil {
				t.Fatalf("users table not found: %v", err)
			}

		})
	}

}
