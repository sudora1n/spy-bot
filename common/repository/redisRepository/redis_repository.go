package redisRepository

import (
	"context"
	"fmt"
	"ssuspy-common/types"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type Redis struct {
	*goredis.Client
}

func NewRedis(cfg *types.RedisConfig) (*Redis, error) {
	client := goredis.NewClient(&goredis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.Database,
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %v", err)
	}

	return &Redis{client}, nil
}
