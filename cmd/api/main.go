package main

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"sample-miniapp-backend/internal/config"
	"sample-miniapp-backend/internal/handlers"
	"sample-miniapp-backend/internal/middleware"
	"sample-miniapp-backend/internal/services"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	redisService, err := services.NewRedisService(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisService.Close()

	jwtService := services.NewJWTService(cfg)

	gameEngine := services.NewGameEngine(redisService)
	wsHandler := handlers.NewWebSocketHandler(gameEngine)

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			gameEngine.CleanupStaleGames(10 * time.Minute)
		}
	}()

	authHandler := handlers.NewAuthHandler(redisService, jwtService, cfg.BotToken)
	userHandler := handlers.NewUserHandler(redisService, gameEngine)
	gameHandler := handlers.NewGameHandler(gameEngine, redisService)

	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	router.GET("/auth/telegram", authHandler.Authenticate)

	protected := router.Group("/api")
	protected.Use(middleware.AuthMiddleware(jwtService))
	{
		protected.GET("/me", userHandler.GetCurrentUser)
		protected.POST("/logout", userHandler.Logout)

		protected.GET("/ws", wsHandler.HandleWebSocket)

		games := protected.Group("/games")
		{
			games.POST("/bet", gameHandler.PlaceBet)
			games.POST("/cashout", gameHandler.Cashout)
			games.GET("/balance", gameHandler.GetBalance)
			games.GET("/active", gameHandler.GetActiveGames)
			games.GET("/history", gameHandler.GetGameHistory)

			games.GET("/verification", gameHandler.GetVerificationData)
			games.POST("/verify", gameHandler.VerifyGame)

			mines := games.Group("/mines")
			{
				mines.POST("/reveal", gameHandler.RevealMine)
				mines.POST("/cashout", gameHandler.CashoutMines)
			}

			dice := games.Group("/dice")
			{
				dice.POST("/play", gameHandler.PlayDice)
			}
		}
	}

	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
