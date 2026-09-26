package services

import (
	"context"
	"errors"
	"strings"

	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/auth"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/models"
	"github.com/google/uuid"
)

const (
	DefaultUserSearchLimit = 20
	MaxUserSearchLimit     = 20
	MinUserSearchLength    = 2
)

type UserRepository interface {
	Create(ctx context.Context, user models.User) (models.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (models.User, error)
	FindByEmail(ctx context.Context, email string) (models.User, error)
	Search(ctx context.Context, query string, limit int) ([]models.User, error)
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

type SearchUsersInput struct {
	Query string
	Limit int
}

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrInvalidUserSearchQuery = errors.New("search query must be at least 2 characters")

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

func (s *UserService) Search(ctx context.Context, input SearchUsersInput) ([]models.User, error) {
	query := strings.TrimSpace(input.Query)
	if len(query) < MinUserSearchLength {
		return nil, ErrInvalidUserSearchQuery
	}

	limit := input.Limit
	if limit <= 0 {
		limit = DefaultUserSearchLimit
	}
	if limit > MaxUserSearchLimit {
		limit = MaxUserSearchLimit
	}

	return s.users.Search(ctx, query, limit)
}
