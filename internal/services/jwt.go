package services

import (
	"fmt"
	"time"

	"sample-miniapp-backend/internal/config"
	"sample-miniapp-backend/internal/models"

	"github.com/golang-jwt/jwt/v5"
)

type JWTService struct {
	secret string
	expiry time.Duration
}

func NewJWTService(cfg *config.Config) *JWTService {
	return &JWTService{
		secret: cfg.JWTSecret,
		expiry: cfg.JWTExpiry,
	}
}

type Claims struct {
	UserID    int64  `json:"user_id"`
	SessionID string `json:"session_id"`
	jwt.RegisteredClaims
}

func (s *JWTService) GenerateToken(userID int64, sessionID string) (*models.AuthResponse, error) {
	expirationTime := time.Now().Add(s.expiry)

	claims := &Claims{
		UserID:    userID,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.secret))
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		Token:   tokenString,
		Expires: expirationTime.Unix(),
	}, nil
}

func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.secret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}
