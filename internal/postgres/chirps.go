package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rjpw/bootdev-chirpy/internal/domain"
	"github.com/rjpw/bootdev-chirpy/internal/postgres/database"
)

func toRepositoryChirp(dbChirp database.Chirp) *domain.Chirp {
	return &domain.Chirp{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID,
	}
}

func (r *Repository) CreateChirp(ctx context.Context, body string, user_id uuid.UUID) (*domain.Chirp, error) {
	now := time.Now().UTC().Truncate(time.Microsecond)
	user, err := r.GetUserByID(ctx, user_id.String())
	if err != nil {
		return nil, mapError(err)
	}
	chirp, err := r.db.CreateChirp(ctx, database.CreateChirpParams{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		Body:      body,
		UserID:    user.ID,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return toRepositoryChirp(chirp), nil
}

func (r *Repository) GetChirpByID(ctx context.Context, id string) (*domain.Chirp, error) {
	return &domain.Chirp{}, nil
}

func (r *Repository) DeleteChirp(ctx context.Context, id string) error          { return nil }
func (r *Repository) DeleteAllChirps(ctx context.Context, user_id string) error { return nil }
