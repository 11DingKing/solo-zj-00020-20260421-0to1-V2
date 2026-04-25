package cache

import (
	"context"
	"fmt"

	"cloud-disk/config"

	"github.com/redis/go-redis/v9"
)

var Redis *redis.Client

func Connect(cfg *config.Config) error {
	Redis = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
	})

	ctx := context.Background()
	if err := Redis.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to redis: %w", err)
	}

	return nil
}
