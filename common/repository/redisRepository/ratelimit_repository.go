package redisRepository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	RATELIMIT_KEY = "telegram_ratelimit"

	DEFAULT_HANDLER_KEY = "handler"
)

type Mode int

const (
	ModeRateLimit Mode = iota
	ModeIsolation
	ModeBoth
)

type LimitConfig struct {
	Mode       Mode
	KeyPrefix  string
	HandlerKey string
	Window     time.Duration
	Limit      int
	QueueSize  int
}

type RateLimitError struct {
	message      string
	pushToLogger bool
	err          error
}

func (e *RateLimitError) Error() string {
	if e.err != nil {
		return fmt.Sprintf("%s: %v", e.message, e.err)
	}
	return e.message
}

func (e *RateLimitError) Unwrap() error {
	return e.err
}

func (e *RateLimitError) PushToLogger() bool {
	return e.pushToLogger
}

func NewRateLimitError(message string, err error, pushToLogger bool) error {
	return &RateLimitError{
		message:      message,
		err:          err,
		pushToLogger: pushToLogger,
	}
}

func (r *Redis) Ratelimit(ctx context.Context, cfg *LimitConfig, userId int64) error {
	baseKey := cfg.KeyPrefix
	if baseKey == "" && (cfg.Mode == ModeRateLimit || cfg.Mode == ModeBoth) {
		baseKey = RATELIMIT_KEY
	}

	handlerKey := cfg.HandlerKey
	if handlerKey == "" {
		handlerKey = DEFAULT_HANDLER_KEY
	}

	queueKey := fmt.Sprintf("%s_queue%s:%d", baseKey, cfg.HandlerKey, userId)
	countKey := fmt.Sprintf("%s:%s:%d", baseKey, cfg.HandlerKey, userId)

	qlen, err := r.LLen(ctx, queueKey).Result()
	if err != nil {
		return NewRateLimitError("redis LLen failed", err, true)
	}
	if qlen >= int64(cfg.QueueSize) {
		return NewRateLimitError("too many requests - queue full", nil, false)
	}

	token := strconv.FormatInt(time.Now().UnixNano(), 10)
	if err := r.RPush(ctx, queueKey, token).Err(); err != nil {
		return NewRateLimitError("redis RPush failed", err, true)
	}

	defer func() {
		if err := r.LRem(ctx, queueKey, 1, token).Err(); err != nil {
			log.Error().Int64("userId", userId).Err(err).Msg("failed to remove token from queue")
		}
	}()

	for {
		head, err := r.LIndex(ctx, queueKey, 0).Result()
		if err != nil {
			return NewRateLimitError("redis LIndex failed", err, true)
		}

		if cfg.Mode == ModeIsolation {
			if head != token {
				time.Sleep(50*time.Millisecond + time.Duration((time.Now().UnixNano()%30))*time.Microsecond) // небольшая джиттер-пауза
				continue
			}
			break
		}

		if head != token {
			time.Sleep(50*time.Millisecond + time.Duration((time.Now().UnixNano()%30))*time.Microsecond)
			continue
		}

		count, err := r.Incr(ctx, countKey).Result()
		if err != nil {
			return NewRateLimitError("redis Incr failed", err, true)
		}
		if count == 1 {
			if err := r.Expire(ctx, countKey, cfg.Window).Err(); err != nil {
				return NewRateLimitError("redis Expire failed", err, true)
			}
		}

		if count > int64(cfg.Limit) {
			ttl, err := r.TTL(ctx, countKey).Result()
			if err != nil {
				return NewRateLimitError("redis TTL failed", err, true)
			}
			if ttl <= 0 {
				ttl = cfg.Window
			}
			time.Sleep(ttl + 10*time.Millisecond)
			continue
		}

		break
	}

	return nil
}
