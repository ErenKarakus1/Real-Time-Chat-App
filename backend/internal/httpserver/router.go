package httpserver

import (
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/config"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/handlers"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/middleware"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/presence"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/realtime"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/repositories"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func NewRouter(cfg config.Config, dbPool *pgxpool.Pool, redisClient *redis.Client) *gin.Engine {
	router := gin.Default()

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

	registerRoutes(router, cfg.JWTSecret, authHandler, conversationHandler, messageHandler, webSocketHandler)

	return router
}

func registerRoutes(
	router *gin.Engine,
	jwtSecret string,
	authHandler *handlers.AuthHandler,
	conversationHandler *handlers.ConversationHandler,
	messageHandler *handlers.MessageHandler,
	webSocketHandler *handlers.WebSocketHandler,
) {
	authMiddleware := middleware.Auth(jwtSecret)

	router.GET("/health", handlers.Health)
	router.POST("/auth/register", authHandler.Register)
	router.POST("/auth/login", authHandler.Login)
	router.GET("/auth/me", authMiddleware, authHandler.Me)
	router.GET("/users/search", authMiddleware, authHandler.Search)
	router.GET("/users/presence", authMiddleware, authHandler.Presence)

	router.GET("/conversations", authMiddleware, conversationHandler.List)
	router.PATCH("/conversations/:conversation_id", authMiddleware, conversationHandler.UpdateRoomName)
	router.DELETE("/conversations/:conversation_id", authMiddleware, conversationHandler.DeleteRoom)
	router.POST("/conversations/rooms", authMiddleware, conversationHandler.CreateRoom)
	router.POST("/conversations/direct", authMiddleware, conversationHandler.CreateDirect)
	router.POST("/conversations/:conversation_id/read", authMiddleware, conversationHandler.MarkRead)
	router.PATCH("/conversations/:conversation_id/owner", authMiddleware, conversationHandler.TransferOwnership)
	router.GET("/conversations/:conversation_id/participants", authMiddleware, conversationHandler.ListParticipants)
	router.POST("/conversations/:conversation_id/participants", authMiddleware, conversationHandler.AddParticipant)
	router.PATCH("/conversations/:conversation_id/participants/:user_id/role", authMiddleware, conversationHandler.UpdateParticipantRole)
	router.DELETE("/conversations/:conversation_id/participants/me", authMiddleware, conversationHandler.LeaveRoom)
	router.DELETE("/conversations/:conversation_id/participants/:user_id", authMiddleware, conversationHandler.RemoveParticipant)

	router.GET("/conversations/:conversation_id/messages", authMiddleware, messageHandler.List)
	router.POST("/conversations/:conversation_id/messages", authMiddleware, messageHandler.Create)
	router.PATCH("/conversations/:conversation_id/messages/:message_id", authMiddleware, messageHandler.Update)
	router.DELETE("/conversations/:conversation_id/messages/:message_id", authMiddleware, messageHandler.Delete)

	router.GET("/ws/conversations/:conversation_id", webSocketHandler.Conversation)
}
