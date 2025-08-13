package middleware

import (
	"errors"
	"ssuspy-common/repository/redisRepository"
	"ssuspy-common/types"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/rs/zerolog"
)

type MiddlewareGroup struct {
	rdb *redisRepository.Redis
}

func NewMiddlewareGroup(rdb *redisRepository.Redis) *MiddlewareGroup {
	return &MiddlewareGroup{
		rdb: rdb,
	}
}

func (h *MiddlewareGroup) ConcurrencyMiddleware(cfg redisRepository.LimitConfig) th.Handler {
	return func(ctx *th.Context, update telego.Update) error {
		log := ctx.Value("log").(*zerolog.Logger)
		internalUser := ctx.Value("internalUser").(*types.InternalUser)

		err := h.rdb.Ratelimit(ctx, &cfg, internalUser.ID)
		if err != nil {
			var rateLimitErr *redisRepository.RateLimitError
			if errors.As(err, &rateLimitErr) && rateLimitErr.PushToLogger() {
				log.Error().Err(err).Msg("ratelimit err")
				return err
			}

			return err
		}

		return ctx.Next(update)
	}
}
