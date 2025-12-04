package models

import "time"

type TelegramUser struct {
	ID           int64  `json:"id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name,omitempty"`
	Username     string `json:"username,omitempty"`
	LanguageCode string `json:"language_code,omitempty"`
	IsPremium    bool   `json:"is_premium,omitempty"`
	PhotoURL     string `json:"photo_url,omitempty"`
}

type UserSession struct {
	TelegramUser
	SessionID    string    `json:"session_id"`
	InitDataHash string    `json:"init_data_hash"`
	CreatedAt    time.Time `json:"created_at"`
	LastAccessed time.Time `json:"last_accessed"`
}

type AuthResponse struct {
	Token   string        `json:"token"`
	Expires int64         `json:"expires"`
	User    *TelegramUser `json:"user"`
}

type InitData struct {
	QueryID  string `form:"query_id"`
	User     string `form:"user"`
	AuthDate string `form:"auth_date"`
	Hash     string `form:"hash"`
}
