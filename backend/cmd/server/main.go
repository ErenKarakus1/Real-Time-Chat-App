package main

import (
	"context"
	"log"

	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/config"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/db"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/handlers"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/middleware"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/realtime"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/repositories"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/services"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	gin.SetMode(cfg.GinMode)
	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	ctx := context.Background()
	dbPool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer dbPool.Close()

	redisClient, err := db.ConnectRedis(ctx, cfg.RedisURL)
	if err != nil {
		log.Fatalf("connect redis: %v", err)
	}
	defer redisClient.Close()

	router := gin.Default()
	userRepository := repositories.NewUserRepository(dbPool)
	userService := services.NewUserService(userRepository, cfg.JWTSecret)
	authHandler := handlers.NewAuthHandler(userService)
	conversationRepository := repositories.NewConversationRepository(dbPool)
	conversationService := services.NewConversationService(conversationRepository)
	conversationHandler := handlers.NewConversationHandler(conversationService)
	messageRepository := repositories.NewMessageRepository(dbPool)
	messageService := services.NewMessageService(messageRepository, conversationRepository)
	realtimePubSub := realtime.NewRedisPubSub(redisClient)
	realtimeHub := realtime.NewHub(realtimePubSub)
	messageHandler := handlers.NewMessageHandler(messageService, realtimeHub)
	webSocketHandler := handlers.NewWebSocketHandler(conversationRepository, realtimeHub, cfg.JWTSecret)
	authMiddleware := middleware.Auth(cfg.JWTSecret)

	router.GET("/health", handlers.Health)
	router.POST("/auth/register", authHandler.Register)
	router.POST("/auth/login", authHandler.Login)
	router.GET("/auth/me", authMiddleware, authHandler.Me)
	router.GET("/users/search", authMiddleware, authHandler.Search)
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

	if err := router.Run(":" + cfg.Port); err != nil {
		panic(err)
	}
}
