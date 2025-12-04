package models

import "time"

type Wallet struct {
	UserID        int64   `json:"user_id" redis:"user_id"`
	Balance       float64 `json:"balance" redis:"balance"`
	LockedBalance float64 `json:"locked_balance" redis:"locked_balance"`
	TotalWagered  float64 `json:"total_wagered" redis:"total_wagered"`
	TotalWon      float64 `json:"total_won" redis:"total_won"`

	// Provably Fair seeds
	ClientSeed string `json:"client_seed" redis:"client_seed"`
	ServerHash string `json:"server_hash" redis:"server_hash"`
	Nonce      int64  `json:"nonce" redis:"nonce"`
}

type TransactionType string

const (
	TransactionTypeBet      TransactionType = "bet"
	TransactionTypeWin      TransactionType = "win"
	TransactionTypeDeposit  TransactionType = "deposit"
	TransactionTypeWithdraw TransactionType = "withdraw"
	TransactionTypeBonus    TransactionType = "bonus"
)

type Transaction struct {
	ID            string          `json:"id" redis:"id"`
	UserID        int64           `json:"user_id" redis:"user_id"`
	Type          TransactionType `json:"type" redis:"type"`
	Amount        float64         `json:"amount" redis:"amount"`
	BalanceBefore float64         `json:"balance_before" redis:"balance_before"`
	BalanceAfter  float64         `json:"balance_after" redis:"balance_after"`
	GameID        string          `json:"game_id,omitempty" redis:"game_id,omitempty"`
	Description   string          `json:"description" redis:"description"`
	CreatedAt     time.Time       `json:"created_at" redis:"created_at"`
}

type BalanceResponse struct {
	Balance       float64 `json:"balance"`
	LockedBalance float64 `json:"locked_balance"`
	TotalWagered  float64 `json:"total_wagered"`
	TotalWon      float64 `json:"total_won"`
	Available     float64 `json:"available"` // Balance - LockedBalance
}
