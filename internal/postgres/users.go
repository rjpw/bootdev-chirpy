package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rjpw/bootdev-chirpy/internal/domain"
	"github.com/rjpw/bootdev-chirpy/internal/postgres/database"
)

// Compile-time check to ensure that *Repository implements the domain.UserRepository interface.
// This will cause a compilation error if Repository does not satisfy all methods of UserRepository.
var _ domain.UserRepository = (*Repository)(nil)

func toRepositoryUser(dbUser database.User) *domain.User {
	return &domain.User{
		ID:        dbUser.ID,
		Email:     dbUser.Email,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
	}
}

func (s *Repository) CreateUser(ctx context.Context, email string) (*domain.User, error) {
	now := time.Now().UTC().Truncate(time.Microsecond)
	user, err := s.db.CreateUser(ctx, database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		Email:     email,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return toRepositoryUser(user), nil
}

func (s *Repository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	user, err := s.db.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, mapError(err)
	}
	return toRepositoryUser(user), nil
}

func (s *Repository) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	user, err := s.db.GetUserByID(ctx, parsedID)
	if err != nil {
		return nil, mapError(err)
	}
	return toRepositoryUser(user), nil
}

func (s *Repository) DeleteUser(ctx context.Context, email string) error {
	user, err := s.db.GetUserByEmail(ctx, email)
	if err != nil {
		return mapError(err)
	}
	return s.db.DeleteUser(ctx, user.ID)
}

func (s *Repository) UpdateUserEmail(ctx context.Context, oldEmail, newEmail string) error {
	_, err := s.db.GetUserByEmail(ctx, newEmail)
	if err == nil {
		return domain.ErrConflict
	}
	_, err = s.db.GetUserByEmail(ctx, oldEmail)
	if err != nil {
		return mapError(err)
	}
	_, err = s.db.UpdateUser(ctx, database.UpdateUserParams{
		Email:     oldEmail,
		UpdatedAt: time.Now().UTC().Truncate(time.Microsecond),
		Email_2:   newEmail,
	})
	return mapError(err)
}

func (s *Repository) DeleteAllUsers(ctx context.Context) error {
	return s.db.DeleteAllUsers(ctx)
}
