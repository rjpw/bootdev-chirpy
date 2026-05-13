//go:build integration

package postgres_test

/*
Required for ChirpRepository interface:

	CreateChirp(ctx context.Context, body string, user_id uuid.UUID) (*domain.Chirp, error)
	GetChirpByID(ctx context.Context, id string) (*domain.Chirp, error)
	DeleteChirp(ctx context.Context, id string) error
	DeleteAllChirps(ctx context.Context, user_id string) error

*/
