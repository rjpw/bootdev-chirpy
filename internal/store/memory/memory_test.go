package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rjpw/bootdev-chirpy/internal/store"
)

func newStore() store.UserStore {
	return NewMemoryStore()
}

func TestCreateUser(t *testing.T) {
	s := newStore()

	ctx := context.Background()
	email := "test@example.com"
	user, err := s.CreateUser(ctx, email)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	if user.Email != email {
		t.Errorf("Expected email %s, got %s", email, user.Email)
	}
	// check that ID is not zero
	if user.ID == uuid.Nil {
		t.Error("Expected non-zero user ID")
	}
	// check that CreatedAt and UpdatedAt are set and recent
	if user.CreatedAt.IsZero() || user.UpdatedAt.IsZero() {
		t.Error("Expected CreatedAt and UpdatedAt to be set")
	}
	if time.Since(user.CreatedAt) > time.Second*5 {
		t.Error("Expected CreatedAt to be recent")
	}
	if time.Since(user.UpdatedAt) > time.Second*5 {
		t.Error("Expected UpdatedAt to be recent")
	}

}

func TestCreateUserConflict(t *testing.T) {
	s := newStore()

	ctx := context.Background()
	email := "test@example.com"
	_, err := s.CreateUser(ctx, email)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	// try to create another user with the same email,
	// should get conflict error
	_, err = s.CreateUser(ctx, email)
	if !errors.Is(err, store.ErrConflict) {
		t.Fatalf("Expected ErrConflict when creating user with duplicate email, got: %v", err)
	}
}

func TestGetUserByEmail(t *testing.T) {
	s := newStore()

	ctx := context.Background()
	email := "test@example.com"

	_, err := s.CreateUser(ctx, email)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	user, err := s.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("Failed to get user by email: %v", err)
	}
	if user.Email != email {
		t.Errorf("Expected email %s, got %s", email, user.Email)
	}
}

func TestGetUserByEmailNotFound(t *testing.T) {
	s := newStore()

	ctx := context.Background()
	email := "nonexistent@example.com"
	_, err := s.GetUserByEmail(ctx, email)
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("Expected ErrNotFound when getting user by email, got: %v", err)
	}
}
