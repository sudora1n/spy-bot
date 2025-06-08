package middleware

import (
	"context"
	"fmt"
	"runtime/debug"
	"strconv"
	"time"

	commonConsts "github.com/example/current-repo/common/consts"
	commonProm "github.com/example/current-repo/common/prom"
	commonRedis "github.com/example/current-repo/common/redis" // Imports package redisclient
	commonTypes "github.com/example/current-repo/common/types"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// LogPanicHandler recovers from panics, logs them, and increments a Prometheus counter.
func LogPanicHandler(recovered any) error {
	// Ensure commonProm.PanicsTotal is not nil (should be initialized by InitProm)
	if commonProm.PanicsTotal != nil {
		commonProm.PanicsTotal.Inc()
	} else {
		// Fallback logging if prom metrics aren't initialized (should not happen in normal operation)
		log.Error().Msg("commonProm.PanicsTotal is nil, cannot increment panic counter.")
	}


	stack := debug.Stack()
	log.Error().
		Any("recovered", recovered).
		Str("stack", string(stack)).
		Msg("PANIC recovered")

	return fmt.Errorf("panic: %v", recovered)
}

// PromMiddleware measures request duration and counts requests/errors using Prometheus.
func PromMiddleware(c *th.Context, update telego.Update) error {
	handlerNameVal := c.Value("handlerName")
	handlerName, ok := handlerNameVal.(string)
	if !ok {
		log.Warn().Msg("handlerName not found in context or not a string, using 'unknown_handler'")
		handlerName = "unknown_handler"
	}

	start := time.Now()
	if commonProm.RequestsTotal != nil {
		commonProm.RequestsTotal.WithLabelValues(handlerName).Inc()
	} else {
		log.Error().Msg("commonProm.RequestsTotal is nil for PromMiddleware.")
	}


	defer func() {
		duration := time.Since(start).Seconds()
		if commonProm.ProcessingTime != nil {
			commonProm.ProcessingTime.WithLabelValues(handlerName).Observe(duration)
		} else {
			log.Error().Msg("commonProm.ProcessingTime is nil for PromMiddleware.")
		}
	}()

	err := c.Next(update)
	if err != nil {
		if commonProm.ErrorsTotal != nil {
			commonProm.ErrorsTotal.WithLabelValues(handlerName).Inc()
		} else {
			log.Error().Msg("commonProm.ErrorsTotal is nil for PromMiddleware.")
		}
	}

	return err
}

// RateLimitConfig holds configuration for RateLimitMiddleware.
type RateLimitConfig struct {
	CountKeyPrefix string // e.g., "rl_count" or "rl_c_count"
	QueueKeyPrefix string // e.g., "rl_queue" or "rl_c_queue"
	Window         time.Duration
	Limit          int
	QueueSize      int
}

// NewRateLimitMiddleware creates a rate limiting handler.
func NewRateLimitMiddleware(rdb *commonRedis.Redis, cfg RateLimitConfig) th.Handler {
	return func(ctx *th.Context, update telego.Update) error {
		logVal := ctx.Value("log")
		logger, ok := logVal.(*zerolog.Logger)
		if !ok {
			globalLogger := log.With().Logger()
			logger = &globalLogger
			logger.Warn().Msg("Logger not found in context for RateLimitMiddleware, using default global logger.")
		}

		internalUserVal := ctx.Value("internalUser")
		internalUser, ok := internalUserVal.(*commonTypes.InternalUser)
		if !ok || internalUser == nil {
			logger.Error().Msg("InternalUser not found in context or is nil for RateLimitMiddleware")
			return fmt.Errorf("internalUser context missing for rate limiting")
		}

		// Use CountKeyPrefix and QueueKeyPrefix from cfg
		countKey := fmt.Sprintf("%s:%d", cfg.CountKeyPrefix, internalUser.ID)
		queueKey := fmt.Sprintf("%s:%d", cfg.QueueKeyPrefix, internalUser.ID)

		qlen, err := rdb.LLen(context.Background(), queueKey).Result()
		if err != nil {
			logger.Error().Err(err).Str("key", queueKey).Msg("Redis LLen failed for rate limit queue")
			return err
		}
		if qlen >= int64(cfg.QueueSize) {
			logger.Debug().Int64("userId", internalUser.ID).Int64("qlen", qlen).Int("qsize", cfg.QueueSize).Msg("Rate limit queue full")
			return nil
		}

		token := strconv.FormatInt(time.Now().UnixNano(), 10)
		if err := rdb.RPush(context.Background(), queueKey, token).Err(); err != nil {
			logger.Error().Err(err).Str("key", queueKey).Msg("Redis RPush failed for rate limit queue")
			return err
		}

		defer func() {
			if err := rdb.LPop(context.Background(), queueKey).Err(); err != nil {
				logger.Error().Err(err).Str("key", queueKey).Msg("Redis LPop failed for rate limit queue")
			}
		}()

		for {
			head, err := rdb.LIndex(context.Background(), queueKey, 0).Result()
			if err != nil {
				logger.Error().Err(err).Str("key", queueKey).Msg("Redis LIndex failed for rate limit queue")
				return err
			}
			if head != token {
				time.Sleep(50 * time.Millisecond)
				continue
			}

			count, err := rdb.Incr(context.Background(), countKey).Result()
			if err != nil {
				logger.Error().Err(err).Str("key", countKey).Msg("Redis Incr failed for rate limit counter")
				return err
			}
			if count == 1 {
				if err := rdb.Expire(context.Background(), countKey, cfg.Window).Err(); err != nil {
					logger.Error().Err(err).Str("key", countKey).Msg("Redis Expire failed for rate limit counter")
					// Non-fatal, continue processing
				}
			}

			if count > int64(cfg.Limit) {
				ttl, err := rdb.TTL(context.Background(), countKey).Result()
				if err != nil {
					logger.Error().Err(err).Str("key", countKey).Msg("Redis TTL failed for rate limit counter")
					// Potentially stuck if TTL fails; sleep for a default duration
					time.Sleep(cfg.Window / 10)
					continue
				}
				if ttl > 0 {
					time.Sleep(ttl)
				} else {
					time.Sleep(cfg.Window / 10)
				}
				continue
			}
			break
		}
		return ctx.Next(update)
	}
}
