package main

import (
	"context"
	"log"

	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/config"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/db"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/handlers"
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

	router := gin.Default()
	userRepository := repositories.NewUserRepository(dbPool)
	userService := services.NewUserService(userRepository, cfg.JWTSecret)
	authHandler := handlers.NewAuthHandler(userService)

	router.GET("/health", handlers.Health)
	router.POST("/auth/register", authHandler.Register)
	router.POST("/auth/login", authHandler.Login)

	if err := router.Run(":" + cfg.Port); err != nil {
		panic(err)
	}
}
