package services

import (
	"context"
	"errors"
	"testing"

	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

type fakeUserRepository struct {
	createErr error
}

func (r *fakeUserRepository) Create(ctx context.Context, user models.User) (models.User, error) {
	if r.createErr != nil {
		return models.User{}, r.createErr
	}

	return user, nil
}

func (r *fakeUserRepository) FindByID(ctx context.Context, id uuid.UUID) (models.User, error) {
	return models.User{}, nil
}

func (r *fakeUserRepository) FindByEmail(ctx context.Context, email string) (models.User, error) {
	return models.User{}, nil
}

func (r *fakeUserRepository) Search(ctx context.Context, query string, limit int) ([]models.User, error) {
	return nil, nil
}

func TestRegisterReturnsEmailAlreadyRegisteredForUniqueViolation(t *testing.T) {
	service := NewUserService(&fakeUserRepository{
		createErr: &pgconn.PgError{Code: "23505"},
	}, "test-secret")

	_, err := service.Register(context.Background(), RegisterUserInput{
		Username: "eren",
		Email:    "eren@example.com",
		Password: "password123",
	})

	if !errors.Is(err, ErrEmailAlreadyRegistered) {
		t.Fatalf("err = %v, want %v", err, ErrEmailAlreadyRegistered)
	}
}
