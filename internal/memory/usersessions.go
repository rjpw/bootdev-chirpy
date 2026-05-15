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

func (repo *Repository) CreateSession(
	_ context.Context,
	userID uuid.UUID,
) (*domain.UserSession, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	_, err := repo.getUserByID(userID)
	if err != nil {
		return nil, err
	}

	id := auth.MakeRefreshToken()
	now := time.Now().UTC().Truncate(time.Microsecond)

	session := domain.UserSession{
		ID:        id,
		UserID:    userID,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(60 * 24 * time.Hour),
	}

	repo.userSessions[id] = session
	return &session, nil
}

func (repo *Repository) GetSession(_ context.Context, id string) (*domain.UserSession, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	session, ok := repo.userSessions[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &session, nil
}

func (repo *Repository) RevokeSession(ctx context.Context, id string) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	session, err := repo.GetSession(ctx, id)
	if err != nil {
		return err
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	session.RevokedAt = now
	session.UpdatedAt = now

	return nil
}

func (repo *Repository) DeleteSessionsByUserID(_ context.Context, userID uuid.UUID) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	for id, session := range repo.userSessions {
		if session.UserID == userID {
			delete(repo.userSessions, id)
		}
	}

	return nil // is there an error condition in this implementation?
}
