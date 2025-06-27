package redis

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"ssuspy-bot/consts"
	"time"

	"github.com/google/uuid"
)

type PublicDataGifts struct {
	MaxEmission int
}

func (r *Redis) SetPublicDataGifts(ctx context.Context, userID int64, data *PublicDataGifts) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(data); err != nil {
		return "", err
	}

	id := uuid.New().String()

	return id, r.Set(ctx, fmt.Sprintf("%s:%d:%s", consts.REDIS_PUBLIC_GIFTS, userID, id), buf.Bytes(), consts.REDIS_TTL_PUBLIC_GIFTS).Err()
}

func (r *Redis) GetPublicDataGifts(ctx context.Context, userID int64, dataID string) (*PublicDataGifts, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	val, err := r.Get(ctx, fmt.Sprintf("%s:%d:%s", consts.REDIS_PUBLIC_GIFTS, userID, dataID)).Bytes()
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(val)
	dec := gob.NewDecoder(buf)
	var data PublicDataGifts
	err = dec.Decode(&data)

	return &data, err
}

func (r *Redis) UpdatePublicDataGifts(ctx context.Context, userID int64, dataID string, data *PublicDataGifts) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(data); err != nil {
		return err
	}

	key := fmt.Sprintf("%s:%d:%s", consts.REDIS_PUBLIC_GIFTS, userID, dataID)

	return r.Set(ctx, key, buf.Bytes(), consts.REDIS_TTL_PUBLIC_GIFTS).Err()
}

func (r *Redis) RemovePublicDataGifts(ctx context.Context, userID int64, dataID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	key := fmt.Sprintf("%s:%d:%s", consts.REDIS_PUBLIC_GIFTS, userID, dataID)
	return r.Del(ctx, key).Err()
}
