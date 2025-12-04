package services_test

import (
	"testing"
	"time"

	"sample-miniapp-backend/internal/config"
	"sample-miniapp-backend/internal/models"
	"sample-miniapp-backend/internal/services"
)

func TestRedisService(t *testing.T) {
	cfg := &config.Config{
		RedisURL:  "localhost:6379",
		RedisPass: "",
		RedisDB:   0,
	}

	redisService, err := services.NewRedisService(cfg)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer redisService.Close()

	userID := int64(999999)

	wallet, err := redisService.GetWallet(userID)
	if err != nil {
		t.Fatalf("Failed to get wallet: %v", err)
	}

	if wallet.Balance != 10000 {
		t.Errorf("Expected default balance 10000, got %f", wallet.Balance)
	}

	betAmount := 1000.0
	if err := redisService.LockBalanceForGame(userID, betAmount); err != nil {
		t.Errorf("Failed to lock balance: %v", err)
	}

	wallet, err = redisService.GetWallet(userID)
	if err != nil {
		t.Fatalf("Failed to get wallet after lock: %v", err)
	}

	if wallet.Balance != 9000 {
		t.Errorf("Expected balance 9000 after lock, got %f", wallet.Balance)
	}

	if wallet.LockedBalance != 1000 {
		t.Errorf("Expected locked balance 1000, got %f", wallet.LockedBalance)
	}

	session := &models.GameSession{
		ID:         "test_game_123",
		UserID:     userID,
		GameType:   models.GameTypeCrash,
		BetAmount:  betAmount,
		Multiplier: 1.0,
		Status:     "active",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := redisService.SaveGameSession(session); err != nil {
		t.Errorf("Failed to save game session: %v", err)
	}

	retrieved, err := redisService.GetGameSession("test_game_123")
	if err != nil {
		t.Errorf("Failed to get game session: %v", err)
	}

	if retrieved.ID != session.ID {
		t.Errorf("Game session ID mismatch: expected %s, got %s", session.ID, retrieved.ID)
	}

	allowed, err := redisService.CheckRateLimit(userID, "bet", 5, time.Minute)
	if err != nil {
		t.Errorf("Failed to check rate limit: %v", err)
	}

	if !allowed {
		t.Error("First bet should be allowed")
	}

	redisService.DeleteWallet(userID)
	redisService.DeleteGameSession(session.ID)
	redisService.ClearBetRateLimit(userID)
}
