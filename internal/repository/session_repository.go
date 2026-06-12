package repository

import (
	"context"
	"shop_project_be/internal/domain"
	"time"

	"github.com/bytedance/sonic"
	"github.com/redis/go-redis/v9"
)

// onlinePrefix adalah prefix key Redis untuk penanda user online.
const onlinePrefix = "online:"

type sessionRepository struct {
	rdb *redis.Client
}

func NewSessionRepository(rdb *redis.Client) domain.SessionRepository {
	return &sessionRepository{rdb: rdb}
}

// SetUserOnline menandai user online. Key online:<userID> diberi TTL = masa
// berlaku token, jadi otomatis kedaluwarsa bila user tidak logout.
func (s *sessionRepository) SetUserOnline(ctx context.Context, user domain.OnlineUser, ttl time.Duration) error {
	user.LastSeen = time.Now()
	data, err := sonic.Marshal(user)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, onlinePrefix+user.UserID, data, ttl).Err()
}

// RemoveUserOnline menghapus penanda online (dipakai saat logout).
func (s *sessionRepository) RemoveUserOnline(ctx context.Context, userID string) error {
	return s.rdb.Del(ctx, onlinePrefix+userID).Err()
}

// ListOnlineUsers mengumpulkan semua penanda online yang masih hidup di Redis.
func (s *sessionRepository) ListOnlineUsers(ctx context.Context) ([]domain.OnlineUser, error) {
	users := make([]domain.OnlineUser, 0)

	var keys []string
	iter := s.rdb.Scan(ctx, 0, onlinePrefix+"*", 100).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return users, nil
	}

	vals, err := s.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}
	for _, v := range vals {
		str, ok := v.(string)
		if !ok {
			continue // key kedaluwarsa di antara SCAN dan MGET
		}
		var u domain.OnlineUser
		if err := sonic.Unmarshal([]byte(str), &u); err != nil {
			continue
		}
		users = append(users, u)
	}
	return users, nil
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
