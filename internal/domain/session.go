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

// OnlineUser represents a single online user (cashier). Stored in Redis with
// TTL = access token lifetime, so it disappears automatically when the token
// expires even if the user does not log out.
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

	// Presence (who is currently online).
	SetUserOnline(ctx context.Context, user OnlineUser, ttl time.Duration) error
	RemoveUserOnline(ctx context.Context, userID string) error
	ListOnlineUsers(ctx context.Context) ([]OnlineUser, error)
}
