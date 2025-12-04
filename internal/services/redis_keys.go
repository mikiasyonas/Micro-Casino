package services

import "time"

const (
	KeyUserSession        = "user:%d:session:%s"
	KeyUserInfo           = "user:%d:info"
	KeyWallet             = "wallet:%d"
	KeyGameSession        = "game:session:%s"
	KeyUserActiveGames    = "user:%d:active_games"
	KeyUserCompletedGames = "user:%d:completed_games"
	KeyTransaction        = "transaction:%s"
	KeyUserTransactions   = "user:%d:transactions"
	KeyRateLimit          = "ratelimit:%d:%s"
	KeyBetPatterns        = "patterns:%d:bets"

	TTLUserSession = 24 * time.Hour
	TTLUserInfo    = 30 * 24 * time.Hour // 30 days
	TTLGameSession = 7 * 24 * time.Hour  // 7 days
	TTLTransaction = 30 * 24 * time.Hour // 30 days

	DefaultRateLimitBets    = 30 // Max 30 bets per minute
	DefaultRateLimitCashout = 60 // Max 60 cashouts per minute
)
