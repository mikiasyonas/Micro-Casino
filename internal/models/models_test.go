package models_test

import (
	"sample-miniapp-backend/internal/models"
	"testing"
)

func TestModels(t *testing.T) {
	session := &models.GameSession{
		ID:         models.GenerateGameID(),
		UserID:     123456789,
		GameType:   models.GameTypeCrash,
		BetAmount:  1000, // $10.00
		Multiplier: 1.0,
		Status:     "active",
	}

	if session.ID == "" {
		t.Error("GameSession ID should not be empty")
	}

	betReq := &models.BetRequest{
		GameType: models.GameTypeCrash,
		Amount:   50, // $0.50
	}

	if err := betReq.Validate(); err != nil {
		t.Errorf("BetRequest validation failed: %v", err)
	}

	invalidBet := &models.BetRequest{
		GameType: "invalid",
		Amount:   0,
	}

	if err := invalidBet.Validate(); err == nil {
		t.Error("Invalid bet should fail validation")
	}

	wallet, err := models.NewWallet(123456789)
	if err != nil {
		t.Errorf("Failed to create wallet: %v", err)
	}

	if wallet.Balance != 10000 {
		t.Errorf("Expected starting balance 10000, got %f", wallet.Balance)
	}

	if wallet.ClientSeed == "" {
		t.Error("Wallet should have a client seed")
	}
}
