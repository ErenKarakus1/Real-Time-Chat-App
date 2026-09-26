package main

import (
	"context"
	"log"

	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/config"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/db"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/httpserver"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	gin.SetMode(cfg.GinMode)
	if err := cfg.ValidateServer(); err != nil {
		log.Fatal(err)
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

	router := httpserver.NewRouter(cfg, dbPool, redisClient)

	if err := router.Run(":" + cfg.Port); err != nil {
		panic(err)
	}
}
