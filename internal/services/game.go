package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"time"

	"sample-miniapp-backend/internal/models"

	"github.com/google/uuid"
)

type GameEngine struct {
	redisService *RedisService
	serverSeed   string
	activeGames  map[string]*GameInstance
}

type GameInstance struct {
	Session    *models.GameSession
	StartedAt  time.Time
	LastUpdate time.Time
	IsRunning  bool
	StopChan   chan struct{}
}

func NewGameEngine(redisService *RedisService) *GameEngine {
	return &GameEngine{
		redisService: redisService,
		serverSeed:   generateServerSeed(),
		activeGames:  make(map[string]*GameInstance),
	}
}

func generateServerSeed() string {
	// In production, will use use crypto/rand and store securely
	// Rotate this seed every 10,000 games or daily
	bytes := make([]byte, 32)
	rand.Read(bytes) // will be using crypto/rand in production
	return hex.EncodeToString(bytes)
}

func (ge *GameEngine) GetServerHash() string {
	hash := sha256.Sum256([]byte(ge.serverSeed))
	return hex.EncodeToString(hash[:])
}

func (ge *GameEngine) GetServerSpeed() string {
	return ge.serverSeed
}

func (ge *GameEngine) generateCrashPoint(clientSeed string, nonce int64) float64 {
	message := fmt.Sprintf("%s:%d", clientSeed, nonce)
	h := hmac.New(sha256.New, []byte(ge.serverSeed))
	h.Write([]byte(message))
	hash := hex.EncodeToString(h.Sum(nil))

	// Standard crash game formula:
	// Use first 52 bits (13 hex characters) of hash
	hashPrefix := hash[:13]
	n := new(big.Int)
	n.SetString(hashPrefix, 16)

	randFloat := float64(n.Int64()) / math.Pow(2, 52)

	// Calculate crash point with house edge
	// Common formula: e = 0.99 (1% house edge)
	// crashPoint = floor(100 * (1 - e) / (1 - randFloat))
	houseEdge := 0.01 // 1% house edge
	crashPoint := math.Floor(100*(1-houseEdge)/(1-randFloat)) / 100.0

	// Ensure minimum 1.00x and maximum 1000x
	if crashPoint < 1.0 {
		crashPoint = 1.0
	}
	if crashPoint > 1000.0 {
		crashPoint = 1000.0
	}

	return crashPoint
}

// VerifyGameResult allows players to verify game fairness
func (ge *GameEngine) VerifyGameResult(clientSeed, serverSeed string, nonce int64) (float64, string, error) {
	// Recalculate the hash
	message := fmt.Sprintf("%s:%d", clientSeed, nonce)
	h := hmac.New(sha256.New, []byte(serverSeed))
	h.Write([]byte(message))
	calculatedHash := hex.EncodeToString(h.Sum(nil))

	// Generate crash point from hash
	hashPrefix := calculatedHash[:13]
	n := new(big.Int)
	n.SetString(hashPrefix, 16)

	randFloat := float64(n.Int64()) / math.Pow(2, 52)
	houseEdge := 0.01
	crashPoint := math.Floor(100*(1-houseEdge)/(1-randFloat)) / 100.0

	if crashPoint < 1.0 {
		crashPoint = 1.0
	}
	if crashPoint > 1000.0 {
		crashPoint = 1000.0
	}

	return crashPoint, calculatedHash, nil
}

// GetVerificationData returns data needed for client verification
func (ge *GameEngine) GetVerificationData(userID int64) (*models.VerificationData, error) {
	wallet, err := ge.redisService.GetWallet(userID)
	if err != nil {
		return nil, err
	}

	return &models.VerificationData{
		ClientSeed:   wallet.ClientSeed,
		ServerHash:   ge.GetServerHash(),
		CurrentNonce: wallet.Nonce,
	}, nil
}

func (ge *GameEngine) PlaceBet(ctx context.Context, userID int64, req *models.BetRequest) (*models.GameSession, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid bet: %v", err)
	}

	allowed, err := ge.redisService.CheckRateLimit(userID, "bet", 30, time.Minute)
	if err != nil {
		return nil, fmt.Errorf("rate limit check failed: %v", err)
	}
	if !allowed {
		return nil, fmt.Errorf("bet rate limit exceeded")
	}

	wallet, err := ge.redisService.GetWallet(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet: %v", err)
	}

	if wallet.Balance < req.Amount {
		return nil, fmt.Errorf("insufficient balance: have %.2f, need %.2f",
			wallet.Balance, req.Amount)
	}

	if err := ge.redisService.LockBalanceForGame(userID, req.Amount); err != nil {
		return nil, fmt.Errorf("failed to lock balance: %v", err)
	}

	ge.redisService.RecordBetPattern(userID, req.Amount, req.GameType)

	var session *models.GameSession
	switch req.GameType {
	case models.GameTypeCrash:
		session, err = ge.createCrashGame(userID, req.Amount)
	case models.GameTypeMines:
		session, err = ge.createMinesGame(userID, req.Amount)
	case models.GameTypeDice:
		session, err = ge.createDiceGame(userID, req.Amount)
	default:
		ge.redisService.ReleaseBalanceFromGame(userID, req.Amount, false, 0)
		return nil, fmt.Errorf("game type not yet implemented: %s", req.GameType)
	}

	if err != nil {
		ge.redisService.ReleaseBalanceFromGame(userID, req.Amount, false, 0)
		return nil, err
	}

	if err := ge.startGame(session); err != nil {
		ge.redisService.ReleaseBalanceFromGame(userID, req.Amount, false, 0)
		return nil, fmt.Errorf("failed to start game: %v", err)
	}

	return session, nil
}

func (ge *GameEngine) createCrashGame(userID int64, betAmount float64) (*models.GameSession, error) {
	wallet, err := ge.redisService.GetWallet(userID)
	if err != nil {
		return nil, err
	}

	crashPoint := ge.generateCrashPoint(wallet.ClientSeed, wallet.Nonce)

	message := fmt.Sprintf("%s:%d", wallet.ClientSeed, wallet.Nonce)
	h := hmac.New(sha256.New, []byte(ge.serverSeed))
	h.Write([]byte(message))
	gameHash := hex.EncodeToString(h.Sum(nil))

	session := &models.GameSession{
		ID:         uuid.New().String(),
		UserID:     userID,
		GameType:   models.GameTypeCrash,
		BetAmount:  betAmount,
		Multiplier: 1.0,
		CrashPoint: crashPoint,
		ClientSeed: wallet.ClientSeed,
		ServerHash: ge.GetServerHash(),
		ServerSeed: ge.serverSeed,
		Nonce:      wallet.Nonce,
		FinalHash:  gameHash,
		Status:     "active",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := ge.redisService.SaveGameSession(session); err != nil {
		return nil, err
	}

	wallet.Nonce++
	if err := ge.redisService.SaveWallet(wallet); err != nil {
		return nil, err
	}

	return session, nil
}

func (ge *GameEngine) startGame(session *models.GameSession) error {
	gameInstance := &GameInstance{
		Session:    session,
		StartedAt:  time.Now(),
		LastUpdate: time.Now(),
		IsRunning:  true,
		StopChan:   make(chan struct{}),
	}

	ge.activeGames[session.ID] = gameInstance

	switch session.GameType {
	case models.GameTypeCrash:
		go ge.runCrashGame(gameInstance)
	case models.GameTypeMines:
		go ge.runMinesGame(gameInstance)
	// case models.GameTypeDice:
	// 	go ge.runDiceGame(gameInstance)
	default:
		return fmt.Errorf("unsupported game type: %s", session.GameType)
	}

	return nil
}

func (ge *GameEngine) runCrashGame(instance *GameInstance) {
	ticker := time.NewTicker(100 * time.Millisecond) // 10 updates per second
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			instance.Session.Multiplier += 0.01
			instance.Session.UpdatedAt = time.Now()

			ge.redisService.UpdateGameSession(instance.Session)

			if instance.Session.Multiplier >= instance.Session.CrashPoint {
				ge.handleCrash(instance)
				return
			}

		case <-instance.StopChan:
			return
		}
	}
}

func (ge *GameEngine) handleCrash(instance *GameInstance) {
	instance.Session.Status = "crashed"
	instance.Session.EndedAt = time.Now()
	instance.IsRunning = false

	ge.redisService.UpdateGameSession(instance.Session)
	ge.redisService.CompleteGameSession(instance.Session.UserID, instance.Session.ID)

	ge.redisService.ReleaseBalanceFromGame(
		instance.Session.UserID,
		instance.Session.BetAmount,
		false, // lost
		0,     // no winnings
	)

	ge.recordTransaction(instance.Session, false, 0)

	delete(ge.activeGames, instance.Session.ID)
	close(instance.StopChan)
}

func (ge *GameEngine) Cashout(ctx context.Context, userID int64, gameID string) (*models.GameResult, error) {
	allowed, err := ge.redisService.CheckRateLimit(userID, "cashout", 60, time.Minute)
	if err != nil || !allowed {
		return nil, fmt.Errorf("cashout rate limit exceeded")
	}

	instance, exists := ge.activeGames[gameID]
	if !exists {
		session, err := ge.redisService.GetGameSession(gameID)
		if err != nil {
			return nil, fmt.Errorf("game not found")
		}

		if session.Status != "active" {
			return nil, fmt.Errorf("game already ended")
		}

		return nil, fmt.Errorf("game not active")
	}

	if instance.Session.UserID != userID {
		return nil, fmt.Errorf("unauthorized cashout attempt")
	}

	if instance.IsRunning {
		instance.StopChan <- struct{}{}
		instance.IsRunning = false
	}

	winnings := instance.Session.BetAmount * instance.Session.Multiplier

	instance.Session.CashoutAt = instance.Session.Multiplier
	instance.Session.Status = "cashed_out"
	instance.Session.EndedAt = time.Now()
	instance.Session.UpdatedAt = time.Now()

	ge.redisService.UpdateGameSession(instance.Session)
	ge.redisService.CompleteGameSession(userID, gameID)

	err = ge.redisService.ReleaseBalanceFromGame(
		userID,
		instance.Session.BetAmount,
		true,                                // won
		winnings-instance.Session.BetAmount, // net winnings
	)

	if err != nil {
		instance.Session.Status = "active"
		ge.redisService.UpdateGameSession(instance.Session)
		return nil, fmt.Errorf("failed to process cashout: %v", err)
	}

	ge.recordTransaction(instance.Session, true, winnings)

	wallet, _ := ge.redisService.GetWallet(userID)

	delete(ge.activeGames, gameID)
	close(instance.StopChan)

	return &models.GameResult{
		GameID:     gameID,
		Win:        true,
		Multiplier: instance.Session.Multiplier,
		Payout:     winnings,
		NewBalance: wallet.Balance,
	}, nil
}

func (ge *GameEngine) createMinesGame(userID int64, betAmount float64) (*models.GameSession, error) {
	wallet, err := ge.redisService.GetWallet(userID)
	if err != nil {
		return nil, err
	}

	minePositions := ge.generateMinePositions(wallet.ClientSeed, wallet.Nonce)

	session := &models.GameSession{
		ID:         uuid.New().String(),
		UserID:     userID,
		GameType:   models.GameTypeMines,
		BetAmount:  betAmount,
		Multiplier: 1.0,
		ClientSeed: wallet.ClientSeed,
		ServerHash: ge.GetServerHash(),
		Nonce:      wallet.Nonce,
		Status:     "active",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	session.Metadata = map[string]interface{}{
		"mines":       minePositions,
		"grid_size":   25,
		"mine_count":  3,
		"revealed":    []int{},
		"multipliers": ge.calculateMineMultipliers(),
	}

	if err := ge.redisService.SaveGameSession(session); err != nil {
		return nil, err
	}

	wallet.Nonce++
	ge.redisService.SaveWallet(wallet)

	return session, nil
}

func (ge *GameEngine) generateMinePositions(clientSeed string, nonce int64) []int {
	message := fmt.Sprintf("mines:%s:%d", clientSeed, nonce)
	h := hmac.New(sha256.New, []byte(ge.serverSeed))
	h.Write([]byte(message))
	hash := hex.EncodeToString(h.Sum(nil))

	positions := make([]int, 0, 3)
	used := make(map[int]bool)

	for i := 0; i < 3 && len(hash) >= 2; i++ {
		val := int(hash[i*2])*16 + int(hash[i*2+1])
		pos := val % 25

		for used[pos] {
			pos = (pos + 1) % 25
		}

		positions = append(positions, pos)
		used[pos] = true
	}

	return positions
}

func (ge *GameEngine) calculateMineMultipliers() map[int]float64 {
	return map[int]float64{
		0:  1.0,   // 0 mines revealed (impossible)
		1:  1.12,  // 1 mine revealed
		2:  1.3,   // 2 mines revealed
		3:  1.62,  // 3 mines revealed
		4:  2.08,  // 4 mines revealed
		5:  2.85,  // 5 mines revealed
		6:  4.14,  // 6 mines revealed
		7:  6.5,   // 7 mines revealed
		8:  11.5,  // 8 mines revealed
		9:  24.0,  // 9 mines revealed
		10: 75.0,  // 10 mines revealed
		11: 750.0, // 11 mines revealed
	}
}

// runMinesGame runs the mines game (turn-based, not real-time)
func (ge *GameEngine) runMinesGame(instance *GameInstance) {
	// Mines is turn-based, no continuous loop needed
	// Game state managed through API calls
}

func (ge *GameEngine) createDiceGame(userID int64, betAmount float64) (*models.GameSession, error) {
	wallet, err := ge.redisService.GetWallet(userID)
	if err != nil {
		return nil, err
	}

	roll := ge.generateDiceRoll(wallet.ClientSeed, wallet.Nonce)

	session := &models.GameSession{
		ID:         uuid.New().String(),
		UserID:     userID,
		GameType:   models.GameTypeDice,
		BetAmount:  betAmount,
		Multiplier: 1.0,
		ClientSeed: wallet.ClientSeed,
		ServerHash: ge.GetServerHash(),
		Nonce:      wallet.Nonce,
		Status:     "active",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	session.Metadata = map[string]interface{}{
		"roll":    roll,
		"target":  50, // Default target (under 50 wins)
		"is_over": false,
	}

	if err := ge.redisService.SaveGameSession(session); err != nil {
		return nil, err
	}

	wallet.Nonce++
	ge.redisService.SaveWallet(wallet)

	return session, nil
}

func (ge *GameEngine) generateDiceRoll(clientSeed string, nonce int64) int {
	message := fmt.Sprintf("dice:%s:%d", clientSeed, nonce)
	h := hmac.New(sha256.New, []byte(ge.serverSeed))
	h.Write([]byte(message))
	hash := hex.EncodeToString(h.Sum(nil))

	val := int(hash[0])
	return val % 100
}

func (ge *GameEngine) GetActiveGame(gameID string) (*GameInstance, bool) {
	instance, exists := ge.activeGames[gameID]
	return instance, exists
}

func (ge *GameEngine) GetUserActiveGames(userID int64) ([]*models.GameSession, error) {
	gameIDs, err := ge.redisService.GetUserActiveGames(userID)
	if err != nil {
		return nil, err
	}

	var sessions []*models.GameSession
	for _, gameID := range gameIDs {
		session, err := ge.redisService.GetGameSession(gameID)
		if err == nil && session.Status == "active" {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

func (ge *GameEngine) ForceCrash(gameID string) error {
	instance, exists := ge.activeGames[gameID]
	if !exists {
		return fmt.Errorf("game not active")
	}

	ge.handleCrash(instance)
	return nil
}

func (ge *GameEngine) recordTransaction(session *models.GameSession, won bool, payout float64) error {
	txType := models.TransactionTypeBet
	description := fmt.Sprintf("Placed bet on %s", session.GameType)

	if won {
		txType = models.TransactionTypeWin
		description = fmt.Sprintf("Won %.2f on %s (%.2fx)",
			payout, session.GameType, session.Multiplier)
	}

	wallet, err := ge.redisService.GetWallet(session.UserID)
	if err != nil {
		return err
	}

	tx := &models.Transaction{
		ID:            uuid.New().String(),
		UserID:        session.UserID,
		Type:          txType,
		Amount:        payout,
		BalanceBefore: wallet.Balance - payout,
		BalanceAfter:  wallet.Balance,
		GameID:        session.ID,
		Description:   description,
		CreatedAt:     time.Now(),
	}

	return ge.redisService.SaveTransaction(tx)
}

func (ge *GameEngine) CleanupStaleGames(maxAge time.Duration) {
	for _, instance := range ge.activeGames {
		if time.Since(instance.LastUpdate) > maxAge {
			ge.handleCrash(instance)
		}
	}
}

func (ge *GameEngine) RotateServerSeed(newSeed string) {
	ge.serverSeed = newSeed
}
