package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rjpw/bootdev-chirpy/internal/application"
	"github.com/rjpw/bootdev-chirpy/internal/domain"
)

var _ application.ChirpRepository = (*Repository)(nil) // compiler guard for ChirpRepo

func (r *Repository) CreateChirp(ctx context.Context, body string, user_id uuid.UUID) (*domain.Chirp, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, err := r.getUserByID(user_id)
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
		UserID:    user_id,
	}

	fmt.Printf("Chirp created: %v\n", chirp)
	r.chirps[id] = chirp

	return &chirp, nil
}

// currently just a stub
func (r *Repository) GetUserChirps(ctx context.Context, user_id string) ([]domain.Chirp, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, err := uuid.Parse(user_id)
	if err != nil {
		return nil, err
	}

	chirps := make([]domain.Chirp, 0, 10)
	return chirps, nil

}

func (r *Repository) GetChirpByID(ctx context.Context, id string) (*domain.Chirp, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	chirpID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	chirp, ok := r.chirps[chirpID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &chirp, nil
}

func (r *Repository) DeleteChirp(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	chirpID, err := uuid.Parse(id)
	if err != nil {
		return err
	}

	delete(r.chirps, chirpID)
	return nil
}

func (r *Repository) DeleteAllChirps(ctx context.Context, user_id string) error {
	return nil
}
