package handlers

import (
	"net/http"

	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/models"
	"github.com/gin-gonic/gin"
)

func Health(c *gin.Context) {
	c.JSON(http.StatusOK, models.HealthResponse{
		Status: "ok",
	})
}
