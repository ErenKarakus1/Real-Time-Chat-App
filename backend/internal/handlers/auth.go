package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/models"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/services"
	"github.com/gin-gonic/gin"
)

type UserService interface {
	Register(ctx context.Context, input services.RegisterUserInput) (models.User, error)
	Login(ctx context.Context, input services.LoginInput) (services.LoginResult, error)
}

type AuthHandler struct {
	users UserService
}

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	User  models.UserResponse `json:"user"`
	Token string              `json:"token"`
}

func NewAuthHandler(users UserService) *AuthHandler {
	return &AuthHandler{
		users: users,
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if validationError := validateRegisterRequest(req); validationError != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": validationError})
		return
	}

	user, err := h.users.Register(c, services.RegisterUserInput{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not register user"})
		return
	}

	c.JSON(http.StatusCreated, models.NewUserResponse(user))
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if validationError := validateLoginRequest(req); validationError != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": validationError})
		return
	}

	result, err := h.users.Login(c, services.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	c.JSON(http.StatusOK, loginResponse{
		User:  models.NewUserResponse(result.User),
		Token: result.Token,
	})
}

func validateRegisterRequest(req registerRequest) string {
	username := strings.TrimSpace(req.Username)
	email := strings.TrimSpace(req.Email)

	if username == "" {
		return "username is required"
	}

	if len(username) < 3 {
		return "username must be at least 3 characters"
	}

	if email == "" {
		return "email is required"
	}

	if !strings.Contains(email, "@") {
		return "email must be valid"
	}

	if req.Password == "" {
		return "password is required"
	}

	if len(req.Password) < 8 {
		return "password must be at least 8 characters"
	}

	return ""
}

func validateLoginRequest(req loginRequest) string {
	email := strings.TrimSpace(req.Email)

	if email == "" {
		return "email is required"
	}

	if !strings.Contains(email, "@") {
		return "email must be valid"
	}

	if req.Password == "" {
		return "password is required"
	}

	return ""
}
