package models

type GameType string

const (
	GameTypeCrash    GameType = "crash"
	GameTypeMines    GameType = "mines"
	GameTypeCoinFlip GameType = "coinflip"
	GameTypeAviator  GameType = "aviator"
)

type GameSession struct {
	ID           string   `json:"id" redis:"id"`
	UserID       int64    `json:"user_id" redis:"user_id"`
	GameType     GameType `json:"game_type" redis:"game_type"`
	BetAmount    float64  `json:"bet_amount" redis:"bet_amount"`
	Multiplier   float64  `json:"multiplier" redis:"multiplier"`
	CrashPoint   float64  `json:"crash_point" redis:"crash_point"`
	CashoutPoint float64  `json:"cashout_point" redis:"cashout_point"`

	ClientSeed     string `json:"client_seed" redis:"client_seed"`
	ServerSeedHash string `json:"server_seed_hash" redis:"server_seed_hash"`
	Nonce          int64  `json:"nonce" redis:"nonce"`
	Hash           string `json:"hash" redis:"hash"`

	Status    string `json:"status" redis:"status"` // active, cashed_out, crashed, completed
	StartedAt int64  `json:"started_at" redis:"started_at"`
	EndedAt   int64  `json:"ended_at" redis:"ended_at"`
}
