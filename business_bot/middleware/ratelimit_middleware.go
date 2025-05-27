package middleware

import (
	"context"
	"fmt"
	"ssuspy-bot/consts"
	"ssuspy-bot/types"
	"strconv"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type RateLimitConfig struct {
	Window    time.Duration
	Limit     int
	QueueSize int
}

func (h *MiddlewareGroup) RateLimitMiddleware(cfg RateLimitConfig) th.Handler {
	return func(ctx *th.Context, update telego.Update) error {
		log := ctx.Value("log").(*zerolog.Logger)
		internalUser := ctx.Value("internalUser").(*types.InternalUser)

		countKey := fmt.Sprintf("%s:%d", consts.REDIS_RATELIMIT_COUNT, internalUser.ID)
		queueKey := fmt.Sprintf("%s:%d", consts.REDIS_RATELIMIT_QUEUE, internalUser.ID)

		qlen, err := h.rdb.LLen(context.Background(), queueKey).Result()
		if err != nil {
			return err
		}
		if qlen >= int64(cfg.QueueSize) {
			log.Debug().Msg("too many requests")
			return nil
		}

		token := strconv.FormatInt(time.Now().UnixNano(), 10)
		if err := h.rdb.RPush(context.Background(), queueKey, token).Err(); err != nil {
			return err
		}

		defer func() {
			h.rdb.LPop(context.Background(), queueKey)
		}()

		for {
			head, err := h.rdb.LIndex(context.Background(), queueKey, 0).Result()
			if err != nil {
				return err
			}
			if head != token {
				time.Sleep(50 * time.Millisecond)
				continue
			}

			count, err := h.rdb.Incr(context.Background(), countKey).Result()
			if err != nil {
				return err
			}
			if count == 1 {
				h.rdb.Expire(context.Background(), countKey, cfg.Window)
			}

			if count > int64(cfg.Limit) {
				ttl, err := h.rdb.TTL(context.Background(), countKey).Result()
				if err != nil {
					return err
				}
				time.Sleep(ttl)
				continue
			}

			break
		}

		return ctx.Next(update)
	}
}

func (h *MiddlewareGroup) IsolationMiddleware(key string, queueSize int) th.Handler {
	return func(ctx *th.Context, update telego.Update) error {
		internalUser := ctx.Value("internalUser").(*types.InternalUser)

		queueKey := fmt.Sprintf("%s:%d", key, internalUser.ID)

		qlen, err := h.rdb.LLen(context.Background(), queueKey).Result()
		if err != nil {
			return err
		}
		if qlen >= int64(queueSize) {
			log.Debug().Msg("too many requests")
			return nil
		}

		token := strconv.FormatInt(time.Now().UnixNano(), 10)
		if err := h.rdb.RPush(context.Background(), queueKey, token).Err(); err != nil {
			return err
		}

		defer func() {
			h.rdb.LPop(context.Background(), queueKey)
		}()

		for {
			head, err := h.rdb.LIndex(context.Background(), queueKey, 0).Result()
			if err != nil {
				return err
			}
			if head != token {
				time.Sleep(50 * time.Millisecond)
				continue
			}
			break
		}

		return ctx.Next(update)
	}
}
