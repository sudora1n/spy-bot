package redisRepository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/rs/zerolog"
)

type RateLimitConfig struct {
	Window    time.Duration
	Limit     int64
	QueueSize int64

	COUNT_KEY string
	QUEUE_KEY string
}

func (r *Redis) Ratelimit(log *zerolog.Logger, cfg *RateLimitConfig, userId int64) error {
	countKey := fmt.Sprintf("%s:%d", cfg.COUNT_KEY, userId)
	queueKey := fmt.Sprintf("%s:%d", cfg.QUEUE_KEY, userId)

	qlen, err := r.LLen(context.Background(), queueKey).Result()
	if err != nil {
		return err
	}
	if qlen >= cfg.QueueSize {
		log.Debug().Msg("too many requests")
		return nil
	}

	token := strconv.FormatInt(time.Now().UnixNano(), 10)
	if err := r.RPush(context.Background(), queueKey, token).Err(); err != nil {
		return err
	}

	defer func() {
		r.LPop(context.Background(), queueKey)
	}()

	for {
		head, err := r.LIndex(context.Background(), queueKey, 0).Result()
		if err != nil {
			return err
		}
		if head != token {
			time.Sleep(50 * time.Millisecond)
			continue
		}

		count, err := r.Incr(context.Background(), countKey).Result()
		if err != nil {
			return err
		}
		if count == 1 {
			r.Expire(context.Background(), countKey, cfg.Window)
		}

		if count > cfg.Limit {
			ttl, err := r.TTL(context.Background(), countKey).Result()
			if err != nil {
				return err
			}
			time.Sleep(ttl)
			continue
		}

		break
	}

	return nil
}
