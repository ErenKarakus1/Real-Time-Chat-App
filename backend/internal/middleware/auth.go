package middleware

import (
	"net/http"
	"strings"

	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/auth"
	"github.com/gin-gonic/gin"
)

const (
	ContextUserID = "user_id"
	ContextEmail  = "email"
)

func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenValue := bearerToken(c.GetHeader("Authorization"))
		if tokenValue == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization token is required"})
			return
		}

		claims, err := auth.ParseToken(tokenValue, secret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization token is invalid"})
			return
		}

		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextEmail, claims.Email)
		c.Next()
	}
}

func bearerToken(header string) string {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}

	return parts[1]
}
