package redisclient // Changed from "redis" to "redisclient"

import (
	"context"
	"fmt"
	"time"

	// Assuming RedisConfig will be imported from the common config package
	commonConfig "github.com/example/current-repo/common/config"

	goredis "github.com/redis/go-redis/v9"
)

type Redis struct {
	*goredis.Client
}

func NewRedis(cfg *commonConfig.RedisConfig) (Redis, error) {
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
