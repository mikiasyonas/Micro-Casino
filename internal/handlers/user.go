package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"sample-miniapp-backend/internal/services"
)

type UserHandler struct {
	redisService *services.RedisService
}

func NewUserHandler(redisService *services.RedisService) *UserHandler {
	return &UserHandler{
		redisService: redisService,
	}
}

func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	sessionID, exists := c.Get("session_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Session not found"})
		return
	}

	session, err := h.redisService.GetUserSession(userID.(int64), sessionID.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Session expired or invalid"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": session.TelegramUser,
		"session": gin.H{
			"session_id":    session.SessionID,
			"created_at":    session.CreatedAt,
			"last_accessed": session.LastAccessed,
		},
	})
}

func (h *UserHandler) Logout(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	sessionID, exists := c.Get("session_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Session not found"})
		return
	}

	err := h.redisService.DeleteUserSession(userID.(int64), sessionID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to logout"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}
