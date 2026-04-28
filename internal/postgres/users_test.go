//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rjpw/bootdev-chirpy/internal/domain"
)

func TestCreateUser(t *testing.T) {
	s := setupTestRepository(t)
	ctx := context.Background()
	email := "alice@example.com"
	user, err := s.CreateUser(ctx, email)
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
	s := setupTestRepository(t)
	ctx := context.Background()

	_, err := s.CreateUser(ctx, "dupe@example.com")
	if err != nil {
		t.Fatalf("first CreateUser: %v", err)
	}

	_, err = s.CreateUser(ctx, "dupe@example.com")
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("expected ErrConflict, got: %v", err)
	}
}
