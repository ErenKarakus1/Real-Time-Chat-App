package httpserver

import (
	"time"

	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/config"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/handlers"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/middleware"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/presence"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/ratelimit"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/realtime"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/repositories"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/services"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func NewRouter(cfg config.Config, dbPool *pgxpool.Pool, redisClient *redis.Client) *gin.Engine {
	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.CORSAllowedOrigin},
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	presenceService := presence.NewService(redisClient)
	userRepository := repositories.NewUserRepository(dbPool)
	userService := services.NewUserService(userRepository, cfg.JWTSecret)
	authHandler := handlers.NewAuthHandler(userService, presenceService)

	conversationRepository := repositories.NewConversationRepository(dbPool)
	conversationService := services.NewConversationService(conversationRepository)
	conversationHandler := handlers.NewConversationHandler(conversationService)

	messageRepository := repositories.NewMessageRepository(dbPool)
	messageService := services.NewMessageService(messageRepository, conversationRepository)
	realtimePubSub := realtime.NewRedisPubSub(redisClient)
	realtimeHub := realtime.NewHub(realtimePubSub)
	messageHandler := handlers.NewMessageHandler(messageService, realtimeHub)
	webSocketHandler := handlers.NewWebSocketHandler(conversationRepository, realtimeHub, presenceService, cfg.JWTSecret)

	registerRoutes(router, cfg.JWTSecret, redisClient, authHandler, conversationHandler, messageHandler, webSocketHandler)

	return router
}

func registerRoutes(
	router *gin.Engine,
	jwtSecret string,
	redisClient *redis.Client,
	authHandler *handlers.AuthHandler,
	conversationHandler *handlers.ConversationHandler,
	messageHandler *handlers.MessageHandler,
	webSocketHandler *handlers.WebSocketHandler,
) {
	authMiddleware := middleware.Auth(jwtSecret)
	limiter := ratelimit.NewLimiter(redisClient)
	authRateLimit := middleware.RateLimit(limiter, "auth", 5, time.Minute, middleware.RateLimitByIP)
	apiRateLimit := middleware.RateLimit(limiter, "api", 300, time.Minute, middleware.RateLimitByUser)
	messageRateLimit := middleware.RateLimit(limiter, "messages", 60, time.Minute, middleware.RateLimitByUser)

	router.GET("/health", handlers.Health)
	router.POST("/auth/register", authRateLimit, authHandler.Register)
	router.POST("/auth/login", authRateLimit, authHandler.Login)
	router.GET("/auth/me", authMiddleware, apiRateLimit, authHandler.Me)
	router.GET("/users/search", authMiddleware, apiRateLimit, authHandler.Search)
	router.GET("/users/presence", authMiddleware, apiRateLimit, authHandler.Presence)

	router.GET("/conversations", authMiddleware, apiRateLimit, conversationHandler.List)
	router.PATCH("/conversations/:conversation_id", authMiddleware, apiRateLimit, conversationHandler.UpdateRoomName)
	router.DELETE("/conversations/:conversation_id", authMiddleware, apiRateLimit, conversationHandler.DeleteRoom)
	router.POST("/conversations/rooms", authMiddleware, apiRateLimit, conversationHandler.CreateRoom)
	router.POST("/conversations/direct", authMiddleware, apiRateLimit, conversationHandler.CreateDirect)
	router.POST("/conversations/:conversation_id/read", authMiddleware, apiRateLimit, conversationHandler.MarkRead)
	router.PATCH("/conversations/:conversation_id/owner", authMiddleware, apiRateLimit, conversationHandler.TransferOwnership)
	router.GET("/conversations/:conversation_id/participants", authMiddleware, apiRateLimit, conversationHandler.ListParticipants)
	router.POST("/conversations/:conversation_id/participants", authMiddleware, apiRateLimit, conversationHandler.AddParticipant)
	router.PATCH("/conversations/:conversation_id/participants/:user_id/role", authMiddleware, apiRateLimit, conversationHandler.UpdateParticipantRole)
	router.DELETE("/conversations/:conversation_id/participants/me", authMiddleware, apiRateLimit, conversationHandler.LeaveRoom)
	router.DELETE("/conversations/:conversation_id/participants/:user_id", authMiddleware, apiRateLimit, conversationHandler.RemoveParticipant)

	router.GET("/conversations/:conversation_id/messages", authMiddleware, apiRateLimit, messageHandler.List)
	router.POST("/conversations/:conversation_id/messages", authMiddleware, messageRateLimit, messageHandler.Create)
	router.PATCH("/conversations/:conversation_id/messages/:message_id", authMiddleware, messageRateLimit, messageHandler.Update)
	router.DELETE("/conversations/:conversation_id/messages/:message_id", authMiddleware, messageRateLimit, messageHandler.Delete)

	router.GET("/ws/conversations/:conversation_id", webSocketHandler.Conversation)
}
