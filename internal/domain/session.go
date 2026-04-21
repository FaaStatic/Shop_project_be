package domain

import (
	"context"
	"time"
)

type Session struct {
	UserID       string    `json:"user_id"`
	Email        string    `json:"email"`
	Role         string    `json:"role"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
}

type SessionRepository interface {
	CreateSession(ctx context.Context, session *Session, key string, ttl time.Duration) error
	GetSessionByAccessToken(ctx context.Context, key string) (*Session, error)
	DeleteSessionByAccessToken(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}
