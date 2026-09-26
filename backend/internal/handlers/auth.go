package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/middleware"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/models"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/presence"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type UserService interface {
	Register(ctx context.Context, input services.RegisterUserInput) (models.User, error)
	Login(ctx context.Context, input services.LoginInput) (services.LoginResult, error)
	FindByID(ctx context.Context, id uuid.UUID) (models.User, error)
	Search(ctx context.Context, input services.SearchUsersInput) ([]models.User, error)
}

type AuthHandler struct {
	users    UserService
	presence *presence.Service
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

type presenceResponse struct {
	UserID uuid.UUID `json:"user_id"`
	Online bool      `json:"online"`
}

func NewAuthHandler(users UserService, presence *presence.Service) *AuthHandler {
	return &AuthHandler{
		users:    users,
		presence: presence,
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

func (h *AuthHandler) Me(c *gin.Context) {
	userIDValue, exists := c.Get(middleware.ContextUserID)
	userIDString, ok := userIDValue.(string)
	if !exists || !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authenticated user is required"})
		return
	}

	userID, err := uuid.Parse(userIDString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authenticated user is invalid"})
		return
	}

	user, err := h.users.FindByID(c, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not get current user"})
		return
	}

	c.JSON(http.StatusOK, models.NewUserResponse(user))
}

func (h *AuthHandler) Search(c *gin.Context) {
	limit, ok := positiveIntQuery(c, "limit")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be a positive integer"})
		return
	}

	users, err := h.users.Search(c, services.SearchUsersInput{
		Query: c.Query("q"),
		Limit: limit,
	})
	if err == services.ErrInvalidUserSearchQuery {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not search users"})
		return
	}

	c.JSON(http.StatusOK, models.NewUserResponses(users))
}

func (h *AuthHandler) Presence(c *gin.Context) {
	ids := strings.Split(c.Query("ids"), ",")
	userIDs := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}

		userID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ids must contain valid UUIDs"})
			return
		}

		userIDs = append(userIDs, userID)
	}

	statuses, err := h.presence.Statuses(c, userIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not get user presence"})
		return
	}

	response := make([]presenceResponse, 0, len(userIDs))
	for _, userID := range userIDs {
		response = append(response, presenceResponse{
			UserID: userID,
			Online: statuses[userID],
		})
	}

	c.JSON(http.StatusOK, response)
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

func positiveIntQuery(c *gin.Context, key string) (int, bool) {
	value := c.Query(key)
	if value == "" {
		return 0, true
	}

	limit, err := strconv.Atoi(value)
	if err != nil || limit <= 0 {
		return 0, false
	}

	return limit, true
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
