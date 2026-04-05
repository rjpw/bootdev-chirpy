package database_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rjpw/bootdev-chirpy/internal/database"
	"github.com/rjpw/bootdev-chirpy/internal/testdb"
)

func setupTx(t *testing.T) *database.Queries {
	t.Helper()
	db := testdb.Setup(t)
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	t.Cleanup(func() {
		err := tx.Rollback()
		if err != nil {
			t.Fatalf("rollback tx: %v", err)
		}
	})
	return database.New(tx)
}

func mustCreateUser(t *testing.T, q *database.Queries, email string) (database.User, error) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)
	user, err := q.CreateUser(context.Background(), database.CreateUserParams{
		ID: uuid.New(), CreatedAt: now, UpdatedAt: now, Email: email,
	})
	if err != nil {
		return database.User{}, err
	}
	return user, nil
}

func TestCreateUser(t *testing.T) {
	queries := setupTx(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Microsecond)
	params := database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		Email:     "alice@example.com",
	}

	user, err := queries.CreateUser(ctx, params)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	// Verify all fields round-trip correctly
	if user.ID != params.ID {
		t.Errorf("ID: got %v, want %v", user.ID, params.ID)
	}
	if user.Email != params.Email {
		t.Errorf("Email: got %q, want %q", user.Email, params.Email)
	}
	if !user.CreatedAt.Equal(params.CreatedAt) {
		t.Errorf("CreatedAt: got %v, want %v", user.CreatedAt, params.CreatedAt)
	}
	if !user.UpdatedAt.Equal(params.UpdatedAt) {
		t.Errorf("UpdatedAt: got %v, want %v", user.UpdatedAt, params.UpdatedAt)
	}
}

func TestCreateUserDuplicateEmail(t *testing.T) {
	queries := setupTx(t)
	email := "duplicate@example.com"

	// First insert should succeed
	user, err := mustCreateUser(t, queries, email)
	if err != nil {
		t.Fatalf("mustCreateUser(%q): %v", email, err)
	}
	if user.Email != email {
		t.Errorf("Email: got %q, want %q", user.Email, email)
	}

	// Second insert with same email should fail

	_, err = mustCreateUser(t, queries, email)
	if err == nil {
		t.Fatal("expected error for duplicate email, got nil")
	}

	// Verify it's the right kind of error
	var pgErr *pq.Error
	if errors.As(err, &pgErr) {
		if pgErr.Code != "23505" { // unique_violation
			t.Errorf("expected unique_violation (23505), got %s: %s", pgErr.Code, pgErr.Message)
		}
	} else {
		t.Errorf("expected *pq.Error, got %T: %v", err, err)
	}
}

func TestGetUserByEmail(t *testing.T) {
	cases := []struct {
		name      string
		seedUsers []string // emails to pre-create
		email     string
		wantErr   bool
	}{
		{
			name:      "existing user",
			seedUsers: []string{"exists@example.com"},
			email:     "exists@example.com",
		},
		{
			name:    "nonexistent user",
			email:   "ghost@example.com",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			queries := setupTx(t)
			for _, email := range tc.seedUsers {
				_, err := mustCreateUser(t, queries, email)
				if err != nil {
					t.Fatalf("mustCreateUser(%q): %v", email, err)
				}
			}
			user, err := queries.GetUserByEmail(context.Background(), tc.email)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("GetUserByEmail: %v", err)
			}
			if user.Email != tc.email {
				t.Errorf("Email: got %q, want %q", user.Email, tc.email)
			}
		})
	}
}
