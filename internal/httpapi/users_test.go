package httpapi

// This file contains tests for the API server's user-related endpoints.

import (
	"context"
	"errors"
	"testing"

	"github.com/rjpw/bootdev-chirpy/domain"
)

func TestCreateUser(t *testing.T) {
	s := newTestServer("dev")
	ctx := context.Background()

	email := "test@example.com"
	user, err := s.CreateUser(ctx, email)
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}
	if user.Email != email {
		t.Errorf("Expected email %q, got %q", email, user.Email)
	}
}

func TestCreateUserConflict(t *testing.T) {
	s := newTestServer("dev")
	ctx := context.Background()

	email := "test@example.com"
	_, err := s.CreateUser(ctx, email)
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}
	_, err = s.CreateUser(ctx, email)
	if !errors.Is(err, domain.ErrConflict) {
		t.Errorf("Expected error for duplicate user")
	}
}
