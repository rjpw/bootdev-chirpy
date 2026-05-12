package memory

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rjpw/bootdev-chirpy/internal/application"
	"github.com/rjpw/bootdev-chirpy/internal/auth"
	"github.com/rjpw/bootdev-chirpy/internal/domain"
)

var _ application.UserSessionRepository = (*Repository)(nil)

func (r *Repository) CreateSession(ctx context.Context, user_id uuid.UUID) (*domain.UserSession, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, err := r.getUserByID(user_id)
	if err != nil {
		return nil, err
	}

	id := auth.MakeRefreshToken()
	now := time.Now().UTC().Truncate(time.Microsecond)

	session := domain.UserSession{
		ID:        id,
		UserID:    user_id,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(60 * 24 * time.Hour),
	}

	r.userSessions[id] = session
	return &session, nil
}

func (r *Repository) GetSession(ctx context.Context, id string) (*domain.UserSession, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.userSessions[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &session, nil
}

func (r *Repository) RevokeSession(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	session, err := r.GetSession(ctx, id)
	if err != nil {
		return err
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	session.RevokedAt = now
	session.UpdatedAt = now

	return nil
}

func (r *Repository) DeleteSessionsByUserID(ctx context.Context, user_id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, session := range r.userSessions {
		if session.UserID == user_id {
			delete(r.userSessions, id)
		}
	}

	return nil // is there an error condition in this implementation?
}
