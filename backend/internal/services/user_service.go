package services

import (
	"context"
	"strings"

	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/auth"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/models"
	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, user models.User) (models.User, error)
}

type UserService struct {
	users UserRepository
}

type RegisterUserInput struct {
	Username string
	Email    string
	Password string
}

func NewUserService(users UserRepository) *UserService {
	return &UserService{
		users: users,
	}
}

func (s *UserService) Register(ctx context.Context, input RegisterUserInput) (models.User, error) {
	passwordHash, err := auth.HashPassword(input.Password)
	if err != nil {
		return models.User{}, err
	}

	user := models.User{
		ID:           uuid.New(),
		Username:     strings.TrimSpace(input.Username),
		Email:        strings.TrimSpace(strings.ToLower(input.Email)),
		PasswordHash: passwordHash,
	}

	return s.users.Create(ctx, user)
}
