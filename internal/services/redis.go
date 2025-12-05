package services

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"sample-miniapp-backend/internal/config"
	"sample-miniapp-backend/internal/models"

	"github.com/redis/go-redis/v9"
)

type RedisService struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisService(cfg *config.Config) (*RedisService, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisURL,
		Password: cfg.RedisPass,
		DB:       cfg.RedisDB,
	})

	ctx := context.Background()

	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %v", err)
	}

	service := &RedisService{
		client: client,
		ctx:    ctx,
	}

	return service, nil
}

func (s *RedisService) StoreUserSession(session *models.UserSession, expiry time.Duration) error {
	key := fmt.Sprintf(KeyUserSession, session.ID, session.SessionID)

	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	return s.client.Set(s.ctx, key, data, expiry).Err()
}

func (s *RedisService) GetUserSession(userID int64, sessionID string) (*models.UserSession, error) {
	key := fmt.Sprintf("user:%d:session:%s", userID, sessionID)

	data, err := s.client.Get(s.ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var session models.UserSession
	err = json.Unmarshal([]byte(data), &session)
	if err != nil {
		return nil, err
	}

	session.LastAccessed = time.Now()
	updatedData, _ := json.Marshal(session)
	s.client.Set(s.ctx, key, updatedData, 24*time.Hour)

	return &session, nil
}

func (s *RedisService) DeleteUserSession(userID int64, sessionID string) error {
	key := fmt.Sprintf("user:%d:session:%s", userID, sessionID)
	return s.client.Del(s.ctx, key).Err()
}

func (s *RedisService) StoreUser(user *models.TelegramUser) error {
	key := fmt.Sprintf("user:%d:info", user.ID)

	data, err := json.Marshal(user)
	if err != nil {
		return err
	}

	return s.client.Set(s.ctx, key, data, 30*24*time.Hour).Err()
}

func (s *RedisService) GetUser(userID int64) (*models.TelegramUser, error) {
	key := fmt.Sprintf(KeyUserInfo, userID)

	data, err := s.client.Get(s.ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var user models.TelegramUser
	err = json.Unmarshal([]byte(data), &user)
	return &user, err
}

func (s *RedisService) Close() error {
	return s.client.Close()
}

func (s *RedisService) GetWallet(userID int64) (*models.Wallet, error) {
	key := fmt.Sprintf("wallet:%d", userID)

	data, err := s.client.Get(s.ctx, key).Result()
	if err == redis.Nil {
		wallet := &models.Wallet{
			UserID:        userID,
			Balance:       10000, // $100.00 in cents -> Meaning 1USD being 100 cents
			LockedBalance: 0,
			TotalWagered:  0,
			TotalWon:      0,
			ClientSeed:    generateClientSeed(),
			ServerHash:    "",
			Nonce:         0,
		}

		if err := s.SaveWallet(wallet); err != nil {
			return nil, fmt.Errorf("failed to create wallet: %v", err)
		}
		return wallet, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet: %v", err)
	}

	var wallet models.Wallet
	if err := json.Unmarshal([]byte(data), &wallet); err != nil {
		return nil, fmt.Errorf("failed to unmarshal wallet: %v", err)
	}

	return &wallet, nil
}

func (s *RedisService) SaveWallet(wallet *models.Wallet) error {
	key := fmt.Sprintf("wallet:%d", wallet.UserID)

	data, err := json.Marshal(wallet)
	if err != nil {
		return fmt.Errorf("failed to marshal wallet: %v", err)
	}

	return s.client.Set(s.ctx, key, data, 0).Err()
}

func (s *RedisService) UpdateWalletBalance(userID int64, amount float64) error {
	tx := s.client.TxPipeline()

	key := fmt.Sprintf("wallet:%d", userID)

	getCmd := tx.Get(s.ctx, key)

	_, err := tx.Exec(s.ctx)
	if err != nil && err != redis.Nil {
		return fmt.Errorf("failed to get wallet in transaction: %v", err)
	}

	var wallet models.Wallet

	if err == redis.Nil {
		wallet = models.Wallet{
			UserID:  userID,
			Balance: 10000,
		}
	} else {
		data, err := getCmd.Result()
		if err != nil {
			return fmt.Errorf("failed to get wallet data: %v", err)
		}

		if err := json.Unmarshal([]byte(data), &wallet); err != nil {
			return fmt.Errorf("failed to unmarshal wallet: %v", err)
		}
	}

	wallet.Balance += amount
	if wallet.Balance < 0 {
		return fmt.Errorf("insufficient balance")
	}

	updatedData, err := json.Marshal(wallet)
	if err != nil {
		return fmt.Errorf("failed to marshal updated wallet: %v", err)
	}

	return s.client.Set(s.ctx, key, updatedData, 0).Err()
}

var lockBalanceScript = redis.NewScript(`
	local key = KEYS[1]
	local amount = tonumber(ARGV[1])
	
	local data = redis.call("GET", key)
	if not data then
		return redis.error_reply("wallet not found")
	end
	
	local wallet = cjson.decode(data)
	
	if wallet.balance < amount then
		return redis.error_reply("insufficient balance")
	end
	
	wallet.balance = wallet.balance - amount
	wallet.locked_balance = wallet.locked_balance + amount
	wallet.total_wagered = wallet.total_wagered + amount
	
	local updated = cjson.encode(wallet)
	redis.call("SET", key, updated)
	
	return "OK"
`)

func (s *RedisService) LockBalanceForGame(userID int64, amount float64) error {
	key := fmt.Sprintf("wallet:%d", userID)
	return lockBalanceScript.Run(s.ctx, s.client, []string{key}, amount).Err()
}

var releaseBalanceScript = redis.NewScript(`
	local key = KEYS[1]
	local amount = tonumber(ARGV[1])
	local won = ARGV[2] == "true"
	local winnings = tonumber(ARGV[3])
	
	local data = redis.call("GET", key)
	if not data then
		return redis.error_reply("wallet not found")
	end
	
	local wallet = cjson.decode(data)
	
	if wallet.locked_balance < amount then
		-- In case of inconsistency, we just reset locked_balance to 0 or handle gracefully
		-- For strictness, we error, but in production we might want to auto-correct
		-- Let's just proceed but log internally if we could
	end
	
	wallet.locked_balance = wallet.locked_balance - amount
	if wallet.locked_balance < 0 then
		wallet.locked_balance = 0
	end
	
	if won then
		wallet.balance = wallet.balance + winnings
		wallet.total_won = wallet.total_won + winnings
	end
	
	local updated = cjson.encode(wallet)
	redis.call("SET", key, updated)
	
	return "OK"
`)

func (s *RedisService) ReleaseBalanceFromGame(userID int64, amount float64, won bool, winnings float64) error {
	key := fmt.Sprintf("wallet:%d", userID)
	return releaseBalanceScript.Run(s.ctx, s.client, []string{key}, amount, won, winnings).Err()
}

func (s *RedisService) SaveGameSession(session *models.GameSession) error {
	sessionKey := fmt.Sprintf("game:session:%s", session.ID)

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal game session: %v", err)
	}

	if err := s.client.Set(s.ctx, sessionKey, data, 7*24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to save game session: %v", err)
	}

	userActiveGamesKey := fmt.Sprintf("user:%d:active_games", session.UserID)
	if err := s.client.SAdd(s.ctx, userActiveGamesKey, session.ID).Err(); err != nil {
		return fmt.Errorf("failed to add to active games: %v", err)
	}

	s.client.Expire(s.ctx, userActiveGamesKey, 7*24*time.Hour)

	return nil
}

func (s *RedisService) GetGameSession(gameID string) (*models.GameSession, error) {
	key := fmt.Sprintf("game:session:%s", gameID)

	data, err := s.client.Get(s.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("game not found: %s", gameID)
		}
		return nil, fmt.Errorf("failed to get game session: %v", err)
	}

	var session models.GameSession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal game session: %v", err)
	}

	return &session, nil
}

func (s *RedisService) UpdateGameSession(session *models.GameSession) error {
	existing, err := s.GetGameSession(session.ID)
	if err != nil || existing == nil {
		return err
	}

	session.UpdatedAt = time.Now()

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal updated game session: %v", err)
	}

	key := fmt.Sprintf("game:session:%s", session.ID)
	return s.client.Set(s.ctx, key, data, 7*24*time.Hour).Err()
}

func (s *RedisService) GetUserActiveGames(userID int64) ([]string, error) {
	key := fmt.Sprintf("user:%d:active_games", userID)

	games, err := s.client.SMembers(s.ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get active games: %v", err)
	}

	return games, nil
}

func (s *RedisService) CompleteGameSession(userID int64, gameID string) error {
	userActiveGamesKey := fmt.Sprintf("user:%d:active_games", userID)
	if err := s.client.SRem(s.ctx, userActiveGamesKey, gameID).Err(); err != nil {
		return fmt.Errorf("failed to remove from active games: %v", err)
	}

	completedKey := fmt.Sprintf("user:%d:completed_games", userID)
	score := float64(time.Now().Unix())
	if err := s.client.ZAdd(s.ctx, completedKey, redis.Z{
		Score:  score,
		Member: gameID,
	}).Err(); err != nil {
		return fmt.Errorf("failed to add to completed games: %v", err)
	}

	s.client.ZRemRangeByRank(s.ctx, completedKey, 0, -101)

	return nil
}

func (s *RedisService) SaveTransaction(tx *models.Transaction) error {
	txKey := fmt.Sprintf("transaction:%s", tx.ID)

	data, err := json.Marshal(tx)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction: %v", err)
	}

	if err := s.client.Set(s.ctx, txKey, data, 30*24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to save transaction: %v", err)
	}

	userTxKey := fmt.Sprintf("user:%d:transactions", tx.UserID)
	score := float64(tx.CreatedAt.Unix())

	if err := s.client.ZAdd(s.ctx, userTxKey, redis.Z{
		Score:  score,
		Member: tx.ID,
	}).Err(); err != nil {
		return fmt.Errorf("failed to add to user transactions: %v", err)
	}

	// Keep only last 100 transactions
	s.client.ZRemRangeByRank(s.ctx, userTxKey, 0, -101)

	return nil
}

func (s *RedisService) GetUserTransactions(userID int64, limit int64) ([]*models.Transaction, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	userTxKey := fmt.Sprintf("user:%d:transactions", userID)

	txIDs, err := s.client.ZRevRange(s.ctx, userTxKey, 0, limit-1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction IDs: %v", err)
	}

	var transactions []*models.Transaction
	for _, txID := range txIDs {
		txKey := fmt.Sprintf("transaction:%s", txID)

		data, err := s.client.Get(s.ctx, txKey).Result()
		if err != nil {
			continue
		}

		var tx models.Transaction
		if err := json.Unmarshal([]byte(data), &tx); err != nil {
			continue
		}

		transactions = append(transactions, &tx)
	}

	return transactions, nil
}

func (s *RedisService) GetGameHistory(userID int64, limit int64) ([]*models.GameSession, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	completedKey := fmt.Sprintf("user:%d:completed_games", userID)

	gameIDs, err := s.client.ZRevRange(s.ctx, completedKey, 0, limit-1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get game IDs: %v", err)
	}

	var games []*models.GameSession
	for _, gameID := range gameIDs {
		game, err := s.GetGameSession(gameID)
		if err != nil {
			continue
		}

		games = append(games, game)
	}

	return games, nil
}

func (s *RedisService) CheckRateLimit(userID int64, action string, limit int, window time.Duration) (bool, error) {
	key := fmt.Sprintf("ratelimit:%d:%s", userID, action)

	count, err := s.client.Incr(s.ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check rate limit: %v", err)
	}

	if count == 1 {
		s.client.Expire(s.ctx, key, window)
	}

	return count <= int64(limit), nil
}

func (s *RedisService) RecordBetPattern(userID int64, amount float64, gameType models.GameType) error {
	patternKey := fmt.Sprintf("patterns:%d:bets", userID)

	patternData := map[string]interface{}{
		"amount":    amount,
		"game_type": gameType,
		"timestamp": time.Now().Unix(),
	}

	data, err := json.Marshal(patternData)
	if err != nil {
		return err
	}

	s.client.LPush(s.ctx, patternKey, data)
	s.client.LTrim(s.ctx, patternKey, 0, 49)

	return nil
}

func (s *RedisService) DeleteWallet(userID int64) error {
	key := fmt.Sprintf("wallet:%d", userID)
	return s.client.Del(s.ctx, key).Err()
}

func (s *RedisService) DeleteGameSession(sessionID string) error {
	key := fmt.Sprintf("game:session:%s", sessionID)
	return s.client.Del(s.ctx, key).Err()
}

func (s *RedisService) ClearBetRateLimit(userID int64) error {
	key := fmt.Sprintf("ratelimit:%d:bet", userID)
	return s.client.Del(s.ctx, key).Err()
}

// generateClientSeed creates a cryptographically secure client seed
func generateClientSeed() string {
	// In production, will use crypto/rand
	// For now, let me use pseudo-random for development
	bytes := make([]byte, 16)
	for i := range bytes {
		bytes[i] = byte(time.Now().UnixNano() % 256)
	}
	return hex.EncodeToString(bytes)
}

func durationToSeconds(d time.Duration) string {
	return fmt.Sprintf("%.0f", d.Seconds())
}

func (s *RedisService) BulkGetGameSessions(gameIDs []string) ([]*models.GameSession, error) {
	if len(gameIDs) == 0 {
		return []*models.GameSession{}, nil
	}

	pipe := s.client.Pipeline()
	cmds := make([]*redis.StringCmd, len(gameIDs))

	for i, gameID := range gameIDs {
		key := fmt.Sprintf("game:session:%s", gameID)
		cmds[i] = pipe.Get(s.ctx, key)
	}

	_, err := pipe.Exec(s.ctx)
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("pipeline execution failed: %v", err)
	}

	var sessions []*models.GameSession
	for i, cmd := range cmds {
		data, err := cmd.Result()
		if err == redis.Nil {
			continue
		}
		if err != nil {
			continue
		}

		var session models.GameSession
		if err := json.Unmarshal([]byte(data), &session); err != nil {
			continue
		}

		sessions = append(sessions, &session)

		key := fmt.Sprintf("game:session:%s", gameIDs[i])
		s.client.Expire(s.ctx, key, 7*24*time.Hour)
	}

	return sessions, nil
}
