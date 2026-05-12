package memory

import (
	"context"

	"github.com/google/uuid"
	"github.com/rjpw/bootdev-chirpy/internal/application"
	"github.com/rjpw/bootdev-chirpy/internal/domain"
)

var _ application.UserSessionRepository = (*Repository)(nil)

func (r *Repository) CreateSession(ctx context.Context, user_id uuid.UUID) (*domain.UserSession, error) {
	return nil, nil
}

func (r *Repository) GetSession(ctx context.Context, id string) (*domain.UserSession, error) {
	return nil, nil
}

func (r *Repository) RevokeSession(ctx context.Context, id string) error { return nil }

func (r *Repository) DeleteSessionsByUserID(ctx context.Context, user_id uuid.UUID) error { return nil }
