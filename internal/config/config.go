package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port      string
	Env       string
	JWTSecret string
	JWTExpiry time.Duration
	RedisURL  string
	RedisPass string
	RedisDB   int
	BotToken  string
}

func Load() (*Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	jwtExpiryStr := os.Getenv("JWT_EXPIRY")
	if jwtExpiryStr == "" {
		jwtExpiryStr = "24h"
	}
	jwtExpiry, err := time.ParseDuration(jwtExpiryStr)
	if err != nil {
		jwtExpiry = 24 * time.Hour
	}

	redisDB, _ := strconv.Atoi(os.Getenv("REDIS_DB"))

	return &Config{
		Port:      port,
		Env:       os.Getenv("ENV"),
		JWTSecret: os.Getenv("JWT_SECRET"),
		JWTExpiry: jwtExpiry,
		RedisURL:  os.Getenv("REDIS_URL"),
		RedisPass: os.Getenv("REDIS_PASSWORD"),
		RedisDB:   redisDB,
		BotToken:  os.Getenv("TELEGRAM_BOT_TOKEN"),
	}, nil
}
