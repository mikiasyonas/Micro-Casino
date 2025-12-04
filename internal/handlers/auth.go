package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"sample-miniapp-backend/internal/models"
	"sample-miniapp-backend/internal/services"
	"sample-miniapp-backend/internal/utils"
)

type AuthHandler struct {
	redisService *services.RedisService
	jwtService   *services.JWTService
	botToken     string
}

func NewAuthHandler(redisService *services.RedisService, jwtService *services.JWTService, botToken string) *AuthHandler {
	return &AuthHandler{
		redisService: redisService,
		jwtService:   jwtService,
		botToken:     botToken,
	}
}

func (h *AuthHandler) Authenticate(c *gin.Context) {
	var initData models.InitData
	if err := c.ShouldBindQuery(&initData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid initData"})
		return
	}

	queryStr := c.Request.URL.RawQuery

	valid, err := utils.ValidateTelegramInitData(h.botToken, queryStr)
	if err != nil || !valid {
		log.Println("The error", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Telegram initData"})
		return
	}

	isFresh, err := utils.CheckInitDataAge(queryStr)
	if err != nil || !isFresh {
		log.Println("Second error", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "InitData expired"})
		return
	}

	var telegramUser models.TelegramUser
	if err := json.Unmarshal([]byte(initData.User), &telegramUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user data"})
		return
	}

	sessionID := uuid.New().String()

	userSession := &models.UserSession{
		TelegramUser: telegramUser,
		SessionID:    sessionID,
		InitDataHash: initData.Hash,
		CreatedAt:    time.Now(),
		LastAccessed: time.Now(),
	}

	if err := h.redisService.StoreUser(&telegramUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store user data"})
		return
	}

	if err := h.redisService.StoreUserSession(userSession, 24*time.Hour); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store session"})
		return
	}

	authResponse, err := h.jwtService.GenerateToken(telegramUser.ID, sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	authResponse.User = &telegramUser

	c.JSON(http.StatusOK, authResponse)
}
