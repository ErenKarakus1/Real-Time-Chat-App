package main

import (
	"net/http"

	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/config"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	gin.SetMode(cfg.GinMode)

	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	if err := router.Run(":" + cfg.Port); err != nil {
		panic(err)
	}
}
