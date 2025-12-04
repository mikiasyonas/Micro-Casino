package models

type User struct {
	ID         int64  `json:"id" redis:"id"`
	TelegramID int64  `json:"telegram_id" redis:"telegram_id"`
	Username   string `json:"username" redis:"username"`

	Balance       float64 `json:"balance" redis:"balance"`
	LockedBalance float64 `json:"locked_balance" redis:"locked_balance"`

	ClientSeed     string `json:"client_seed" redis:"client_seed"`
	ServerSeed     string `json:"-" redis:"server_seed"`
	ServerSeedHash string `json:"server_seed_hash" redis:"server_seed_hash"`
	Nonce          int64  `json:"nonce" redis:"nonce"`

	MaxBet         float64 `json:"max_bet" redis:"max_bet"`
	DailyLossLimit float64 `json:"daily_loss_limit" redis:"daily_loss_limit"`
	TotalWagered   float64 `json:"total_wagered" redis:"total_wagered"`
	TotalWon       float64 `json:"total_won" redis:"total_won"`

	CreatedAt int64 `json:"created_at" redis:"created_at"`
	UpdatedAt int64 `json:"updated_at" redis:"updated_at"`
}
