package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"sample-miniapp-backend/internal/models"
	"sample-miniapp-backend/internal/services"
)

type UserHandler struct {
	redisService *services.RedisService
	gameEngine   *services.GameEngine
}

func NewUserHandler(redisService *services.RedisService, gameEngine *services.GameEngine) *UserHandler {
	return &UserHandler{
		redisService: redisService,
		gameEngine:   gameEngine,
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

	wallet, err := h.redisService.GetWallet(userID.(int64))
	if err != nil {
		wallet = &models.Wallet{
			UserID:  userID.(int64),
			Balance: wallet.Balance,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"user": session.TelegramUser,
		"session": gin.H{
			"session_id":    session.SessionID,
			"created_at":    session.CreatedAt,
			"last_accessed": session.LastAccessed,
		},
		"wallet": gin.H{
			"balance":       wallet.Balance,
			"locked":        wallet.LockedBalance,
			"available":     wallet.Balance - wallet.LockedBalance,
			"total_wagered": wallet.TotalWagered,
			"total_won":     wallet.TotalWon,
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
