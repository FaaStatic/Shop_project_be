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

// OnlineUser merepresentasikan satu user (kasir) yang sedang online. Disimpan
// di Redis dengan TTL = masa berlaku access token, sehingga otomatis hilang
// saat token kedaluwarsa walau user tidak logout.
type OnlineUser struct {
	UserID   string    `json:"user_id"`
	Username string    `json:"username"`
	Role     string    `json:"role"`
	LastSeen time.Time `json:"last_seen"`
}

type SessionRepository interface {
	CreateSession(ctx context.Context, session *Session, key string, ttl time.Duration) error
	GetSessionByAccessToken(ctx context.Context, key string) (*Session, error)
	DeleteSessionByAccessToken(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)

	// Presence (siapa yang sedang online).
	SetUserOnline(ctx context.Context, user OnlineUser, ttl time.Duration) error
	RemoveUserOnline(ctx context.Context, userID string) error
	ListOnlineUsers(ctx context.Context) ([]OnlineUser, error)
}
