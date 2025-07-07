package redis

import (
	"context"
	"fmt"
	"time"

	"ssuspy-creator-bot/config"

	goredis "github.com/redis/go-redis/v9"
)

type Redis struct {
	*goredis.Client
}

func NewRedis(cfg *config.RedisConfig) (Redis, error) {
	client := goredis.NewClient(&goredis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.Database,
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return Redis{}, fmt.Errorf("redis connection failed: %v", err)
	}

	return Redis{client}, nil
}
