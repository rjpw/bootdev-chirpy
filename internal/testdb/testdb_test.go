package testdb

import "testing"

func TestSetup(t *testing.T) {
	db := Setup(t)

	// Can we reach the database?
	if err := db.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}

	// Did migrations run?
	var tableName string
	err := db.QueryRow(
		"SELECT table_name FROM information_schema.tables WHERE table_name = $1",
		"users",
	).Scan(&tableName)
	if err != nil {
		t.Fatalf("users table not found: %v", err)
	}
}
