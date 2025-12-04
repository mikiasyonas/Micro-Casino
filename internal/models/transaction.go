package models

type TransactionType string

const (
	TransactionTypeBet      TransactionType = "bet"
	TransactionTypeWin      TransactionType = "win"
	TransactionTypeDeposit  TransactionType = "deposit"
	TransactionTypeWithdraw TransactionType = "withdraw"
)

type Transaction struct {
	ID            string          `json:"id" redis:"id"`
	UserID        int64           `json:"user_id" redis:"user_id"`
	Type          TransactionType `json:"type" redis:"type"`
	Amount        float64         `json:"amount" redis:"amount"`
	BalanceBefore float64         `json:"balance_before" redis:"balance_before"`
	BalanceAfter  float64         `json:"balance_after" redis:"balance_after"`
	GameSessionID string          `json:"game_session_id,omitempty" redis:"game_session_id"`
	Description   string          `json:"description" redis:"description"`
	CreatedAt     int64           `json:"created_at" redis:"created_at"`
}
