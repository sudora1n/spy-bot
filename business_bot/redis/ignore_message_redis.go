package redis

import (
	"context"
	"fmt"
	"slices"
	"ssuspy-bot/consts"
	"time"

	"github.com/rs/zerolog/log"
)

func (r *Redis) IgnoreMessage(ctx context.Context, messageID int, chatID int64) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	pipe := r.Pipeline()

	stringKey := fmt.Sprintf("%d|%d", messageID, chatID)

	pipe.LPush(ctx, consts.REDIS_IGNORE, stringKey)
	pipe.Expire(ctx, consts.REDIS_IGNORE, consts.REDIS_TTL_IGNORE)

	_, err := pipe.Exec(ctx)
	return err
}

func (r *Redis) IsMessageIgnore(ctx context.Context, messageID int, chatID int64) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	messages, err := r.LRange(ctx, consts.REDIS_IGNORE, 0, -1).Result()
	if err != nil {
		return false, err
	}

	stringKey := fmt.Sprintf("%d|%d", messageID, chatID)

	if slices.Contains(messages, stringKey) {
		log.Debug().Str("key", stringKey).Msg("is key in ignore")
		return true, nil
	}

	log.Debug().Str("key", stringKey).Strs("keys", messages).Msg("key not found in ignore")
	return false, nil
}
