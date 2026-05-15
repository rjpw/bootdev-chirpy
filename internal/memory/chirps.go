package memory

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rjpw/bootdev-chirpy/internal/application"
	"github.com/rjpw/bootdev-chirpy/internal/domain"
)

var _ application.ChirpRepository = (*Repository)(nil) // compiler guard for ChirpRepo

func (repo *Repository) CreateChirp(
	_ context.Context,
	body string,
	userID uuid.UUID,
) (*domain.Chirp, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	_, err := repo.getUserByID(userID)
	if err != nil {
		return nil, err
	}

	id := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)
	chirp := domain.Chirp{
		ID:        id,
		CreatedAt: now,
		UpdatedAt: now,
		Body:      body,
		UserID:    userID,
	}

	repo.chirps[id] = chirp
	return &chirp, nil
}

// currently just a stub
func (repo *Repository) GetUserChirps(_ context.Context, userID string) ([]domain.Chirp, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	_, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	chirps := make([]domain.Chirp, 0, 10)
	return chirps, nil
}

func (repo *Repository) GetChirpByID(_ context.Context, id string) (*domain.Chirp, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	chirpID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	chirp, ok := repo.chirps[chirpID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &chirp, nil
}

func (repo *Repository) DeleteChirp(_ context.Context, id string) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	chirpID, err := uuid.Parse(id)
	if err != nil {
		return err
	}

	delete(repo.chirps, chirpID)
	return nil
}

func (repo *Repository) DeleteAllChirps(_ context.Context, _ string) error {
	return nil
}
