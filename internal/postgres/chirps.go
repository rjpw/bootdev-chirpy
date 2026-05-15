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

func toDomainChirps(chirps []database.Chirp) []domain.Chirp {
	var dchirps []domain.Chirp
	for _, c := range chirps {
		dchirps = append(dchirps, *toRepositoryChirp(c))
	}
	return dchirps
}

func (r *Repository) CreateChirp(
	ctx context.Context,
	body string,
	user_id uuid.UUID,
) (*domain.Chirp, error) {
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
	chirpid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	chirp, err := r.db.GetChirpByID(ctx, chirpid)
	if err != nil {
		return nil, mapError(err)
	}

	return toRepositoryChirp(chirp), nil
}

func (r *Repository) GetUserChirps(ctx context.Context, user_id string) ([]domain.Chirp, error) {

	if user_id == "%" || len(user_id) == 0 {

		chirps, err := r.db.GetAllChirps(ctx)
		if err != nil {
			return nil, mapError(err)
		}
		return toDomainChirps(chirps), nil

	} else {

		uid, err := uuid.Parse(user_id)
		if err != nil {
			return nil, mapError(err)
		}

		chirps, err := r.db.GetUserChirps(ctx, uid)
		return toDomainChirps(chirps), nil

	}

}

func (r *Repository) DeleteChirp(ctx context.Context, id string) error {
	chirpid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	err = r.db.DeleteChirpByID(ctx, chirpid)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repository) DeleteAllChirps(ctx context.Context, user_id string) error { return nil }
