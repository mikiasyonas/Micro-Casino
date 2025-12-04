package models

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func GenerateGameID() string {
	return fmt.Sprintf("game_%s_%d",
		time.Now().Format("20060102"),
		uuid.New().ID())
}

func GenerateTransactionID() string {
	return fmt.Sprintf("tx_%s_%d",
		time.Now().Format("20060102"),
		uuid.New().ID())
}

func GenerateClientSeed() (string, error) {
	bytes := make([]byte, 16) // 128 bits of entropy
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate client seed: %v", err)
	}
	return hex.EncodeToString(bytes), nil
}

func (br *BetRequest) Validate() error {
	if br.Amount < 1 {
		return fmt.Errorf("bet amount must be at least 1 cent")
	}
	if br.Amount > 10000 {
		return fmt.Errorf("maximum bet amount is 10000 cents ($100)")
	}

	switch br.GameType {
	case GameTypeCrash, GameTypeMines, GameTypeDice, GameTypeAviator:
	default:
		return fmt.Errorf("invalid game type: %s", br.GameType)
	}

	return nil
}

func CalculatePayout(betAmount, multiplier float64) float64 {
	return betAmount * multiplier
}

func FormatCurrency(cents float64) string {
	return fmt.Sprintf("$%.2f", cents/100)
}

func NewWallet(userID int64) (*Wallet, error) {
	clientSeed, err := GenerateClientSeed()
	if err != nil {
		return nil, err
	}

	return &Wallet{
		UserID:     userID,
		Balance:    10000, // $100.00 starting balance -> This is in cents so 1USD -> is 100 cents
		ClientSeed: clientSeed,
		Nonce:      0,
	}, nil
}
