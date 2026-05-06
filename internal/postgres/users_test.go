//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rjpw/bootdev-chirpy/internal/auth"
	"github.com/rjpw/bootdev-chirpy/internal/domain"
)

func TestCreateUser(t *testing.T) {
	repos := setupTestRepository(t)
	ctx := context.Background()
	email := "alice@example.com"
	password, _ := auth.HashPassword("123456")
	user, err := repos.CreateUser(ctx, email, password)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	// The store should have generated a UUID
	if user.ID.String() == "00000000-0000-0000-0000-000000000000" {
		t.Error("expected non-zero UUID")
	}

	if !user.CreatedAt.After(time.Now().Add(-10 * time.Second)) {
		t.Error("expected recent CreatedAt")
	}
	if !user.UpdatedAt.After(time.Now().Add(-10 * time.Second)) {
		t.Error("expected recent UpdatedAt")
	}
	if user.Email != email {
		t.Errorf("Email: got %q, want %q", user.Email, email)
	}
}

func TestCreateUserDuplicateEmail(t *testing.T) {
	repos := setupTestRepository(t)
	ctx := context.Background()

	password, _ := auth.HashPassword("123456")

	_, err := repos.CreateUser(ctx, "dupe@example.com", password)
	if err != nil {
		t.Fatalf("first CreateUser: %v", err)
	}

	_, err = repos.CreateUser(ctx, "dupe@example.com", password)
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected ErrConflict, got: %v", err)
	}
}
