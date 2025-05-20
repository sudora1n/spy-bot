package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"ssuspy-bot/types"
	"time"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	goredis "github.com/redis/go-redis/v9"
)

type Job struct {
	Loc       *i18n.Localizer
	File      *types.MediaItem
	UserID    int64
	ChatID    int64
	MessageID int
	Caption   string
}

func (r *Redis) EnqueueJob(ctx context.Context, queueKey string, job Job) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}
	return r.RPush(ctx, queueKey, data).Err()
}

func (r *Redis) DequeueJob(ctx context.Context, queueKey string, timeout time.Duration) (*Job, error) {
	res, err := r.BRPop(ctx, timeout, queueKey).Result()
	if err != nil {
		if err == goredis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("redis BRPop error: %w", err)
	}

	var job Job
	if err := json.Unmarshal([]byte(res[1]), &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}
	return &job, nil
}

func (r *Redis) QueueLen(ctx context.Context, queueKey string) (int64, error) {
	return r.LLen(ctx, queueKey).Result()
}
