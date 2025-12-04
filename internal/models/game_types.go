package models

type VerificationData struct {
	ClientSeed   string `json:"client_seed"`
	ServerHash   string `json:"server_hash"`
	CurrentNonce int64  `json:"current_nonce"`
}

type MinesRevealRequest struct {
	GameID   string `json:"game_id" binding:"required"`
	Position int    `json:"position" binding:"required,min=0,max=24"`
}

type MinesRevealResponse struct {
	GameID     string  `json:"game_id"`
	IsMine     bool    `json:"is_mine"`
	Multiplier float64 `json:"multiplier"`
	Positions  []int   `json:"positions"`
	GameOver   bool    `json:"game_over"`
	Winnings   float64 `json:"winnings,omitempty"`
}

type DicePlayRequest struct {
	GameID string `json:"game_id" binding:"required"`
	Target int    `json:"target" binding:"required,min=1,max=95"`
	Over   bool   `json:"over"` // true = over target, false = under target
}

type DicePlayResponse struct {
	GameID     string  `json:"game_id"`
	Roll       int     `json:"roll"`
	Target     int     `json:"target"`
	Win        bool    `json:"win"`
	Multiplier float64 `json:"multiplier"`
	Payout     float64 `json:"payout"`
}
