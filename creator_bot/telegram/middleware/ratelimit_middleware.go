package middleware

import (
	"ssuspy-common/repository/redisRepository"
	"ssuspy-creator-bot/types"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/rs/zerolog"
)

func (h *MiddlewareGroup) RateLimitMiddleware(cfg *redisRepository.RateLimitConfig) th.Handler {
	return func(ctx *th.Context, update telego.Update) error {
		log := ctx.Value("log").(*zerolog.Logger)
		internalUser := ctx.Value("internalUser").(*types.InternalUser)

		err := h.repository.Redis.Ratelimit(log, cfg, internalUser.ID)
		if err != nil {
			return err
		}

		return ctx.Next(update)
	}
}
