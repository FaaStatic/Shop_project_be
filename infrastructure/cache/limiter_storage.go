package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// LimiterStorage adalah implementasi fiber.Storage berbasis Redis. Dipakai oleh
// rate limiter agar hitungan request DIBAGI lintas proses — penting karena
// prefork aktif (tiap child proses kalau pakai store in-memory akan punya
// hitungan sendiri sehingga batas jadi longgar). Dengan Redis, semua proses
// berbagi counter yang sama.
//
// Tiap limiter sebaiknya memakai prefix berbeda supaya counter-nya tidak
// bertabrakan (mis. "rl:login:" vs "rl:global:").
type LimiterStorage struct {
	rdb    *redis.Client
	prefix string
}

// NewLimiterStorage membuat storage dengan prefix key tertentu. Client Redis
// dimiliki oleh pemanggil (tidak ditutup oleh storage ini).
func NewLimiterStorage(rdb *redis.Client, prefix string) *LimiterStorage {
	return &LimiterStorage{rdb: rdb, prefix: prefix}
}

func (s *LimiterStorage) key(k string) string { return s.prefix + k }

func (s *LimiterStorage) GetWithContext(ctx context.Context, key string) ([]byte, error) {
	if key == "" {
		return nil, nil
	}
	val, err := s.rdb.Get(ctx, s.key(key)).Bytes()
	if err == redis.Nil {
		return nil, nil // key tidak ada -> (nil, nil) sesuai kontrak fiber.Storage
	}
	if err != nil {
		return nil, err
	}
	return val, nil
}

func (s *LimiterStorage) Get(key string) ([]byte, error) {
	return s.GetWithContext(context.Background(), key)
}

func (s *LimiterStorage) SetWithContext(ctx context.Context, key string, val []byte, exp time.Duration) error {
	if key == "" || len(val) == 0 {
		return nil
	}
	return s.rdb.Set(ctx, s.key(key), val, exp).Err()
}

func (s *LimiterStorage) Set(key string, val []byte, exp time.Duration) error {
	return s.SetWithContext(context.Background(), key, val, exp)
}

func (s *LimiterStorage) DeleteWithContext(ctx context.Context, key string) error {
	if key == "" {
		return nil
	}
	return s.rdb.Del(ctx, s.key(key)).Err()
}

func (s *LimiterStorage) Delete(key string) error {
	return s.DeleteWithContext(context.Background(), key)
}

// ResetWithContext menghapus semua key milik prefix ini saja (bukan FLUSHDB),
// supaya tidak mengganggu session/presence yang juga tersimpan di Redis.
func (s *LimiterStorage) ResetWithContext(ctx context.Context) error {
	var keys []string
	iter := s.rdb.Scan(ctx, 0, s.prefix+"*", 100).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	return s.rdb.Del(ctx, keys...).Err()
}

func (s *LimiterStorage) Reset() error {
	return s.ResetWithContext(context.Background())
}

// Close tidak menutup client Redis karena dimiliki & ditutup oleh pemanggil
// (cmd). Mengembalikan nil agar aman dipanggil limiter saat shutdown.
func (s *LimiterStorage) Close() error {
	return nil
}
