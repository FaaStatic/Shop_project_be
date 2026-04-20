package cache

import (
	"context"
	"fmt"
	envconfig "shop_project_be/internal/config/env_config"

	"github.com/redis/go-redis/v9"
)

func InitRedis(cfg *envconfig.RedisConfig) (*redis.Client, error) {

	rdb := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.Db,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("Failed COnnet to Redis: %w", err)
	}

	return rdb, nil

}
