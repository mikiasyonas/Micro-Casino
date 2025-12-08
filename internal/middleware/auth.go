package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"sample-miniapp-backend/internal/services"
)

func AuthMiddleware(jwtService *services.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		var tokenString string

		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
				c.Abort()
				return
			}
			tokenString = parts[1]
		} else {
			tokenString = c.Query("token")
			if tokenString == "" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
				c.Abort()
				return
			}
		}

		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("session_id", claims.SessionID)

		c.Next()
	}
}

func RateLimitMiddleware(redisService *services.RedisService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.Next()
			return
		}

		path := c.Request.URL.Path

		var limit int
		var window time.Duration

		switch {
		case strings.Contains(path, "/games/bet"):
			limit = 30 // 30 bets per minute
			window = time.Minute
		case strings.Contains(path, "/games/cashout"):
			limit = 60 // 60 cashouts per minute
			window = time.Minute
		case strings.Contains(path, "/games/mines/reveal"):
			limit = 120 // 120 reveals per minute
			window = time.Minute
		default:
			c.Next()
			return
		}

		allowed, err := redisService.CheckRateLimit(userID.(int64), path, limit, window)
		if err != nil || !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"retry_after": window.Seconds(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
