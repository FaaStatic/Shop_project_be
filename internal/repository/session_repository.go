package repository

import (
	"context"
	"shop_project_be/internal/domain"
	"time"

	"github.com/bytedance/sonic"
	"github.com/redis/go-redis/v9"
)

type sessionRepository struct {
	rdb *redis.Client
}

func NewSessionRepository(rdb *redis.Client) domain.SessionRepository {
	return &sessionRepository{rdb: rdb}
}

// CreateSession implements [domain.SessionRepository].
func (s *sessionRepository) CreateSession(ctx context.Context, session *domain.Session, key string, ttl time.Duration) error {
	data, err := sonic.Marshal(session)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, key, data, ttl).Err()
}

// DeleteSessionByAccessToken implements [domain.SessionRepository].
func (s *sessionRepository) DeleteSessionByAccessToken(ctx context.Context, key string) error {
	return s.rdb.Del(ctx, key).Err()
}

// Exists implements [domain.SessionRepository].
func (s *sessionRepository) Exists(ctx context.Context, key string) (bool, error) {
	exists, err := s.rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// GetSessionByAccessToken implements [domain.SessionRepository].
func (s *sessionRepository) GetSessionByAccessToken(ctx context.Context, key string) (*domain.Session, error) {
	data, err := s.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	var session domain.Session
	err = sonic.Unmarshal([]byte(data), &session)
	if err != nil {
		return nil, err
	}
	return &session, nil
}
