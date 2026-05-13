package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rjpw/bootdev-chirpy/internal/application"
	"github.com/rjpw/bootdev-chirpy/internal/auth"
	"github.com/rjpw/bootdev-chirpy/internal/domain"
	"github.com/rjpw/bootdev-chirpy/internal/postgres/database"
)

var _ application.UserSessionRepository = (*Repository)(nil)

func toDomainUserSession(dbUserSession database.UserSession) *domain.UserSession {
	var revocationTime time.Time

	if dbUserSession.RevokedAt.Valid {
		revocationTime = dbUserSession.RevokedAt.Time
	}

	return &domain.UserSession{
		ID:        dbUserSession.ID,
		UserID:    dbUserSession.UserID,
		CreatedAt: dbUserSession.CreatedAt,
		UpdatedAt: dbUserSession.UpdatedAt,
		ExpiresAt: dbUserSession.ExpiresAt,
		RevokedAt: revocationTime,
	}
}

func (r *Repository) CreateSession(
	ctx context.Context,
	user_id uuid.UUID,
) (*domain.UserSession, error) {
	now := time.Now().UTC().Truncate(time.Microsecond)
	userSession, err := r.db.CreateSession(ctx, database.CreateSessionParams{
		ID:        auth.MakeRefreshToken(),
		UserID:    user_id,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(60 * 24 * time.Hour),
	})
	if err != nil {
		return nil, mapError(err)
	}
	return toDomainUserSession(userSession), nil
}

func (r *Repository) GetSession(ctx context.Context, id string) (*domain.UserSession, error) {
	userSession, err := r.db.GetSession(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	return toDomainUserSession(userSession), nil
}

func (r *Repository) RevokeSession(ctx context.Context, id string) error {
	now := time.Now().UTC().Truncate(time.Microsecond)
	err := r.db.RevokeSession(ctx, database.RevokeSessionParams{
		ID:        id,
		UpdatedAt: now,
	})
	if err != nil {
		return mapError(err)
	}
	return nil
}

func (r *Repository) DeleteSessionsByUserID(ctx context.Context, user_id uuid.UUID) error {
	err := r.db.DeleteSessionsByUserID(ctx, user_id)
	if err != nil {
		return mapError(err)
	}
	return nil
}
