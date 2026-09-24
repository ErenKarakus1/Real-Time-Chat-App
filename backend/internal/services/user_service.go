package services

import (
	"context"
	"errors"
	"strings"

	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/auth"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/models"
	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, user models.User) (models.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (models.User, error)
	FindByEmail(ctx context.Context, email string) (models.User, error)
}

type UserService struct {
	users     UserRepository
	jwtSecret string
}

type RegisterUserInput struct {
	Username string
	Email    string
	Password string
}

type LoginInput struct {
	Email    string
	Password string
}

type LoginResult struct {
	User  models.User
	Token string
}

var ErrInvalidCredentials = errors.New("invalid credentials")

func NewUserService(users UserRepository, jwtSecret string) *UserService {
	return &UserService{
		users:     users,
		jwtSecret: jwtSecret,
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

func (s *UserService) Login(ctx context.Context, input LoginInput) (LoginResult, error) {
	email := strings.TrimSpace(strings.ToLower(input.Email))
	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		return LoginResult{}, ErrInvalidCredentials
	}

	if !auth.CheckPassword(input.Password, user.PasswordHash) {
		return LoginResult{}, ErrInvalidCredentials
	}

	token, err := auth.GenerateToken(user, s.jwtSecret)
	if err != nil {
		return LoginResult{}, err
	}

	return LoginResult{
		User:  user,
		Token: token,
	}, nil
}

func (s *UserService) FindByID(ctx context.Context, id uuid.UUID) (models.User, error) {
	return s.users.FindByID(ctx, id)
}
