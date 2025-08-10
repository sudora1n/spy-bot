package repository

import (
	"context"
	"ssuspy-bot/repository/redis"
	localRedisRepository "ssuspy-bot/repository/redis"
	"ssuspy-common/repository/mongoRepository"
	"ssuspy-common/repository/redisRepository"
	"ssuspy-common/types"

	"github.com/rs/zerolog/log"
)

type Repository struct {
	Mongo  *mongoRepository.MongoRepository
	LRedis *redis.Redis
	CRedis *redisRepository.Redis
}

func NewRepository(
	mongoCfg *types.MongoConfig,
	redisCfg *types.RedisConfig,
) *Repository {
	mongoRepo, err := mongoRepository.NewMongoRepository(mongoCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to MongoDB")
	}

	redis, err := redisRepository.NewRedis(redisCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to common Redis")
	}

	lRedis, err := localRedisRepository.NewRedis(redisCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to local Redis")
	}

	return &Repository{
		Mongo:  mongoRepo,
		CRedis: redis,
		LRedis: lRedis,
	}
}

func (r *Repository) Close(ctx context.Context) error {
	if err := r.CRedis.Close(); err != nil {
		return err
	}
	if err := r.LRedis.Close(); err != nil {
		return err
	}
	return r.Mongo.Disconnect(ctx)
}
