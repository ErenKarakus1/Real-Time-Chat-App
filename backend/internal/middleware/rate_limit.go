package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/ratelimit"
	"github.com/gin-gonic/gin"
)

type RateLimitKeyFunc func(c *gin.Context) string

func RateLimit(limiter *ratelimit.Limiter, name string, limit int64, window time.Duration, keyFunc RateLimitKeyFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := keyFunc(c)
		if key == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authenticated user is required"})
			return
		}

		result, err := limiter.Allow(c, name, key, limit, window)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "could not check rate limit"})
			return
		}

		c.Header("X-RateLimit-Limit", strconv.FormatInt(result.Limit, 10))
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(result.Remaining, 10))

		if !result.Allowed {
			retryAfter := int(result.RetryAfter.Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}

			c.Header("Retry-After", strconv.Itoa(retryAfter))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}

		c.Next()
	}
}

func RateLimitByIP(c *gin.Context) string {
	return c.ClientIP()
}

func RateLimitByUser(c *gin.Context) string {
	userID, ok := c.Get(ContextUserID)
	if !ok {
		return ""
	}

	userIDString, ok := userID.(string)
	if !ok {
		return ""
	}

	return userIDString
}
