package models

import "time"

type GameType string

const (
	GameTypeCrash   GameType = "crash"
	GameTypeMines   GameType = "mines"
	GameTypeDice    GameType = "dice"
	GameTypeAviator GameType = "aviator"
)

type GameSession struct {
	ID         string   `json:"id" redis:"id"`
	UserID     int64    `json:"user_id" redis:"user_id"`
	GameType   GameType `json:"game_type" redis:"game_type"`
	BetAmount  float64  `json:"bet_amount" redis:"bet_amount"`
	Multiplier float64  `json:"multiplier" redis:"multiplier"`
	CashoutAt  float64  `json:"cashout_at" redis:"cashout_at"`
	CrashPoint float64  `json:"crash_point" redis:"crash_point"`

	ClientSeed string `json:"client_seed" redis:"client_seed"`
	ServerSeed string `json:"-" redis:"server_seed"`
	ServerHash string `json:"server_hash" redis:"server_hash"`
	Nonce      int64  `json:"nonce" redis:"nonce"`
	FinalHash  string `json:"final_hash" redis:"final_hash"`

	Status    string                 `json:"status" redis:"status"` // active, cashed_out, crashed, completed
	CreatedAt time.Time              `json:"created_at" redis:"created_at"`
	UpdatedAt time.Time              `json:"updated_at" redis:"updated_at"`
	EndedAt   time.Time              `json:"ended_at" redis:"ended_at"`
	Metadata  map[string]interface{} `json:"metadata" redis:"metadata"`
}

type BetRequest struct {
	GameType GameType `json:"game_type" binding:"required"`
	Amount   float64  `json:"amount" binding:"required,min=1,max=10000"`
}

type CashoutRequest struct {
	GameID string `json:"game_id" binding:"required"`
}

type GameResult struct {
	GameID     string  `json:"game_id"`
	Win        bool    `json:"win"`
	Multiplier float64 `json:"multiplier"`
	Payout     float64 `json:"payout"`
	NewBalance float64 `json:"new_balance"`
}

type GameHistory struct {
	ID         string    `json:"id"`
	GameType   GameType  `json:"game_type"`
	BetAmount  float64   `json:"bet_amount"`
	Multiplier float64   `json:"multiplier"`
	Payout     float64   `json:"payout"`
	Result     string    `json:"result"` // win, lose
	CreatedAt  time.Time `json:"created_at"`
}
