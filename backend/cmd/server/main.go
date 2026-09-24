package main

import (
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/config"
	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/handlers"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	gin.SetMode(cfg.GinMode)

	router := gin.Default()

	router.GET("/health", handlers.Health)

	if err := router.Run(":" + cfg.Port); err != nil {
		panic(err)
	}
}
