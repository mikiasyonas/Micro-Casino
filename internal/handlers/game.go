package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"sample-miniapp-backend/internal/models"
	"sample-miniapp-backend/internal/services"
)

type GameHandler struct {
	gameEngine   *services.GameEngine
	redisService *services.RedisService
}

func NewGameHandler(gameEngine *services.GameEngine, redisService *services.RedisService) *GameHandler {
	return &GameHandler{
		gameEngine:   gameEngine,
		redisService: redisService,
	}
}

func (h *GameHandler) PlaceBet(c *gin.Context) {
	userID := c.GetInt64("user_id")

	var req models.BetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// Rate Limit: 30 bets per minute
	allowed, err := h.redisService.CheckRateLimit(userID, "bet", 30, 1*time.Minute)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Rate limit check failed"})
		return
	}
	if !allowed {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many bets. Please wait."})
		return
	}

	if req.Amount < 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Minimum bet is 1 cent",
		})
		return
	}

	if req.Amount > 10000 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Maximum bet is 10000 cents ($100)",
		})
		return
	}

	session, err := h.gameEngine.PlaceBet(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to place bet",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"game": gin.H{
			"id":          session.ID,
			"game_type":   session.GameType,
			"bet_amount":  session.BetAmount,
			"multiplier":  session.Multiplier,
			"server_hash": session.ServerHash,
			"nonce":       session.Nonce,
			"client_seed": session.ClientSeed,
			"crash_point": session.CrashPoint,
			"status":      session.Status,
			"created_at":  session.CreatedAt,
		},
	})
}

func (h *GameHandler) Cashout(c *gin.Context) {
	userID := c.GetInt64("user_id")

	var req models.CashoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// Rate Limit: 60 cashouts per minute
	allowed, err := h.redisService.CheckRateLimit(userID, "cashout", 60, 1*time.Minute)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Rate limit check failed"})
		return
	}
	if !allowed {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many cashouts. Please wait."})
		return
	}

	result, err := h.gameEngine.Cashout(c.Request.Context(), userID, req.GameID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to cashout",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result":  result,
	})
}

func (h *GameHandler) GetBalance(c *gin.Context) {
	userID := c.GetInt64("user_id")

	wallet, err := h.redisService.GetWallet(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get wallet",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"balance": gin.H{
			"available":     wallet.Balance - wallet.LockedBalance,
			"locked":        wallet.LockedBalance,
			"total":         wallet.Balance,
			"total_wagered": wallet.TotalWagered,
			"total_won":     wallet.TotalWon,
			"nonce":         wallet.Nonce,
			"client_seed":   wallet.ClientSeed,
			"server_hash":   wallet.ServerHash,
		},
	})
}

func (h *GameHandler) GetActiveGames(c *gin.Context) {
	userID := c.GetInt64("user_id")

	games, err := h.gameEngine.GetUserActiveGames(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch active games",
			"details": err.Error(),
		})
		return
	}

	var response []gin.H
	for _, game := range games {
		response = append(response, gin.H{
			"id":          game.ID,
			"game_type":   game.GameType,
			"bet_amount":  game.BetAmount,
			"multiplier":  game.Multiplier,
			"crash_point": game.CrashPoint,
			"cashout_at":  game.CashoutAt,
			"status":      game.Status,
			"created_at":  game.CreatedAt,
			"updated_at":  game.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"games":   response,
		"count":   len(response),
	})
}

func (h *GameHandler) GetGameHistory(c *gin.Context) {
	userID := c.GetInt64("user_id")

	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 50
	}

	games, err := h.redisService.GetGameHistory(userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get game history",
			"details": err.Error(),
		})
		return
	}

	var response []gin.H
	for _, game := range games {
		result := "lose"
		payout := 0.0

		if game.Status == "cashed_out" && game.CashoutAt > 0 {
			result = "win"
			payout = game.BetAmount * game.CashoutAt
		}

		response = append(response, gin.H{
			"id":         game.ID,
			"game_type":  game.GameType,
			"bet_amount": game.BetAmount,
			"multiplier": game.CashoutAt,
			"payout":     payout,
			"result":     result,
			"status":     game.Status,
			"created_at": game.CreatedAt,
			"ended_at":   game.EndedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"games":   response,
		"count":   len(response),
	})
}

func (h *GameHandler) GetVerificationData(c *gin.Context) {
	userID := c.GetInt64("user_id")

	// Get verification data from GameEngine
	// Note: You need to add this method to GameEngine
	// data, err := h.gameEngine.GetVerificationData(userID)

	// For now, get wallet data
	wallet, err := h.redisService.GetWallet(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get verification data",
			"details": err.Error(),
		})
		return
	}

	serverHash := h.gameEngine.GetServerHash()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"client_seed":   wallet.ClientSeed,
			"server_hash":   serverHash,
			"current_nonce": wallet.Nonce,
			"user_id":       userID,
		},
	})
}

func (h *GameHandler) VerifyGame(c *gin.Context) {
	var req struct {
		ClientSeed string `json:"client_seed" binding:"required"`
		ServerSeed string `json:"server_seed" binding:"required"`
		Nonce      int64  `json:"nonce" binding:"required"`
		GameType   string `json:"game_type" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	crashPoint, hash, err := h.gameEngine.VerifyGameResult(
		req.ClientSeed,
		req.ServerSeed,
		req.Nonce,
	)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Verification failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"verification": gin.H{
			"valid":           true,
			"crash_point":     crashPoint,
			"calculated_hash": hash,
			"game_type":       req.GameType,
			"client_seed":     req.ClientSeed,
			"server_seed":     req.ServerSeed,
			"nonce":           req.Nonce,
		},
	})
}

func (h *GameHandler) RevealMine(c *gin.Context) {
	userID := c.GetInt64("user_id")

	var req struct {
		GameID   string `json:"game_id" binding:"required"`
		Position int    `json:"position" binding:"required,min=0,max=24"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// Rate Limit: 120 reveals per minute
	allowed, err := h.redisService.CheckRateLimit(userID, "reveal", 120, 1*time.Minute)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Rate limit check failed"})
		return
	}
	if !allowed {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many reveals. Please wait."})
		return
	}

	session, err := h.redisService.GetGameSession(req.GameID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Game not found",
			"details": err.Error(),
		})
		return
	}

	if session.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "You don't own this game",
		})
		return
	}

	if session.Status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "Game is not active",
			"status": session.Status,
		})
		return
	}

	metadata := session.Metadata

	minePositionsRaw, ok := metadata["mines"]
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Mine data missing",
		})
		return
	}

	minePositions := make([]int, 0)
	if mines, ok := minePositionsRaw.([]interface{}); ok {
		for _, m := range mines {
			if pos, ok := m.(float64); ok {
				minePositions = append(minePositions, int(pos))
			}
		}
	}

	revealedRaw, _ := metadata["revealed"]
	revealed := make([]int, 0)
	if rev, ok := revealedRaw.([]interface{}); ok {
		for _, r := range rev {
			if pos, ok := r.(float64); ok {
				revealed = append(revealed, int(pos))
			}
		}
	}

	for _, pos := range revealed {
		if pos == req.Position {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Position already revealed",
			})
			return
		}
	}

	isMine := false
	for _, minePos := range minePositions {
		if minePos == req.Position {
			isMine = true
			break
		}
	}

	revealed = append(revealed, req.Position)
	metadata["revealed"] = revealed

	multipliers := map[int]float64{
		0:  1.0,
		1:  1.12,
		2:  1.3,
		3:  1.62,
		4:  2.08,
		5:  2.85,
		6:  4.14,
		7:  6.5,
		8:  11.5,
		9:  24.0,
		10: 75.0,
		11: 750.0,
	}

	revealedCount := len(revealed)
	multiplier := multipliers[revealedCount]

	if isMine {
		session.Status = "lost"
		session.EndedAt = time.Now()

		h.redisService.ReleaseBalanceFromGame(
			userID,
			session.BetAmount,
			false, // lost
			0,     // no winnings
		)

		wallet, err := h.redisService.GetWallet(userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":  "Wallet error active",
				"status": session.Status,
			})
		}

		h.redisService.SaveTransaction(&models.Transaction{
			ID:            generateTransactionID(),
			UserID:        userID,
			Type:          models.TransactionTypeBet,
			Amount:        -session.BetAmount,
			BalanceBefore: wallet.Balance,
			BalanceAfter:  0,
			GameID:        session.ID,
			Description:   fmt.Sprintf("Lost mines game at position %d", req.Position),
			CreatedAt:     time.Now(),
		})
	} else {
		session.Multiplier = multiplier
		session.Metadata = metadata
	}

	h.redisService.UpdateGameSession(session)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	response := gin.H{
		"game_id":        session.ID,
		"is_mine":        isMine,
		"position":       req.Position,
		"multiplier":     multiplier,
		"revealed":       revealed,
		"revealed_count": revealedCount,
		"mines_left":     len(minePositions),
		"game_over":      isMine,
		"status":         session.Status,
	}

	if isMine {
		response["mine_positions"] = minePositions
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result":  response,
	})
	})
}

func (h *GameHandler) CashoutMines(c *gin.Context) {
	userID := c.GetInt64("user_id")

	var req struct {
		GameID string `json:"game_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	session, err := h.redisService.GetGameSession(req.GameID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Game not found",
			"details": err.Error(),
		})
		return
	}

	if session.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "You don't own this game",
		})
		return
	}

	// Check game status
	if session.Status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "Game is not active",
			"status": session.Status,
		})
		return
	}

	metadata := session.Metadata

	revealedRaw, _ := metadata["revealed"]
	revealedCount := 0
	if rev, ok := revealedRaw.([]interface{}); ok {
		revealedCount = len(rev)
	}

	multipliers := map[int]float64{
		0:  1.0,
		1:  1.12,
		2:  1.3,
		3:  1.62,
		4:  2.08,
		5:  2.85,
		6:  4.14,
		7:  6.5,
		8:  11.5,
		9:  24.0,
		10: 75.0,
		11: 750.0,
	}

	multiplier := multipliers[revealedCount]
	winnings := session.BetAmount * multiplier

	session.Status = "cashed_out"
	session.CashoutAt = multiplier
	session.Multiplier = multiplier
	session.EndedAt = time.Now()

	err = h.redisService.ReleaseBalanceFromGame(
		userID,
		session.BetAmount,
		true,                       // won
		winnings-session.BetAmount, // net winnings
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to process cashout",
			"details": err.Error(),
		})
		return
	}

	h.redisService.UpdateGameSession(session)
	h.redisService.CompleteGameSession(userID, req.GameID)

	wallet, err := h.redisService.GetWallet(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "Wallet error active",
			"status": session.Status,
		})
	}

	h.redisService.SaveTransaction(&models.Transaction{
		ID:            generateTransactionID(),
		UserID:        userID,
		Type:          models.TransactionTypeWin,
		Amount:        winnings,
		BalanceBefore: wallet.Balance,
		BalanceAfter:  0,
		GameID:        session.ID,
		Description:   fmt.Sprintf("Mines cashout at %.2fx with %d reveals", multiplier, revealedCount),
		CreatedAt:     time.Now(),
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result": gin.H{
			"game_id":        session.ID,
			"multiplier":     multiplier,
			"bet_amount":     session.BetAmount,
			"winnings":       winnings,
			"revealed_count": revealedCount,
			"new_balance":    wallet.Balance,
			"status":         session.Status,
		},
	})
}

func (h *GameHandler) PlayDice(c *gin.Context) {
	userID := c.GetInt64("user_id")

	var req struct {
		GameID string `json:"game_id" binding:"required"`
		Target int    `json:"target" binding:"required,min=1,max=95"`
		Over   bool   `json:"over"` // true = over target, false = under target
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// Rate Limit: 30 bets per minute (same as PlaceBet)
	allowed, err := h.redisService.CheckRateLimit(userID, "bet", 30, 1*time.Minute)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Rate limit check failed"})
		return
	}
	if !allowed {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many bets. Please wait."})
		return
	}

	result, err := h.gameEngine.PlayDice(c.Request.Context(), userID, req.GameID, req.Target, req.Over)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to play dice",
			"details": err.Error(),
		})
		return
	}

	wallet, err := h.redisService.GetWallet(userID)
	if err != nil {
		// Should not happen as PlayDice succeeds
		log.Printf("Failed to get wallet after dice play: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result": gin.H{
			"game_id":     result.GameID,
			"roll":        result.Roll,
			"target":      result.Target,
			"over":        req.Over,
			"win":         result.Win,
			"multiplier":  result.Multiplier,
			"bet_amount":  0, // TODO: Add BetAmount to response model if needed by frontend
			"payout":      result.Payout,
			"new_balance": wallet.Balance,
			"status":      "completed",
		},
	})
}

func generateTransactionID() string {
	return fmt.Sprintf("tx_%d", time.Now().UnixNano())
}
