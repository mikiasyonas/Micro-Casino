package services_test

import (
	"context"
	"log"
	"math"
	"testing"
	"time"

	"sample-miniapp-backend/internal/config"
	"sample-miniapp-backend/internal/models"
	"sample-miniapp-backend/internal/services"
)

func TestGameEngine(t *testing.T) {
	redisService := setupTestRedis(t)
	gameEngine := services.NewGameEngine(redisService)

	ctx := context.Background()
	userID := int64(123456)

	betReq := &models.BetRequest{
		GameType: models.GameTypeCrash,
		Amount:   1000,
	}

	session, err := gameEngine.PlaceBet(ctx, userID, betReq)
	if err != nil {
		t.Fatalf("Failed to place bet: %v", err)
	}

	if session.ID == "" {
		t.Error("Session should have an ID")
	}

	if session.CrashPoint < 1.0 || session.CrashPoint > 1000.0 {
		t.Errorf("Crash point should be between 1.0 and 1000.0, got %.2f", session.CrashPoint)
	}

	time.Sleep(100 * time.Millisecond)

	result, err := gameEngine.Cashout(ctx, userID, session.ID)
	if err != nil {
		t.Fatalf("Failed to cashout: %v", err)
	}

	if !result.Win {
		t.Error("Cashout should result in win")
	}

	if result.Multiplier < 1.0 {
		t.Errorf("Multiplier should be at least 1.0, got %.2f", result.Multiplier)
	}

	crashPoint, _, err := gameEngine.VerifyGameResult(
		session.ClientSeed,
		gameEngine.GetServerSpeed(),
		session.Nonce,
	)

	if err != nil {
		t.Errorf("Verification failed: %v", err)
	}

	if math.Abs(crashPoint-session.CrashPoint) > 0.01 {
		t.Errorf("Verification mismatch: expected %.2f, got %.2f",
			session.CrashPoint, crashPoint)
	}

	cleanupTestData(t, redisService, userID, session.ID)
}

func setupTestRedis(t *testing.T) *services.RedisService {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	redisService, err := services.NewRedisService(cfg)
	if err != nil {
		t.Fatalf("Failed to set up Redis service: %v", err)
	}
	return redisService
}

func cleanupTestData(t *testing.T, redisService *services.RedisService, userID int64, gameID string) {
	err := redisService.DeleteUserSession(userID, gameID)
	if err != nil {
		t.Errorf("Failed to cleanup user data: %v", err)
	}
	err = redisService.DeleteGameSession(gameID)
	if err != nil {
		t.Errorf("Failed to cleanup game data: %v", err)
	}
}
