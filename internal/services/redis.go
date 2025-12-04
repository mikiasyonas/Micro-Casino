package services

import (
	"context"
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

	return &RedisService{
		client: client,
		ctx:    ctx,
	}, nil
}

func (s *RedisService) StoreUserSession(session *models.UserSession, expiry time.Duration) error {
	key := fmt.Sprintf("user:%d:session:%s", session.ID, session.SessionID)

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
	key := fmt.Sprintf("user:%d:info", userID)

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
