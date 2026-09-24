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
}

type AuthHandler struct {
	users UserService
}

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
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
