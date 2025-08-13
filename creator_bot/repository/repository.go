package repository

import (
	"context"
	"ssuspy-common/repository/mongoRepository"
	"ssuspy-common/repository/redisRepository"
	"ssuspy-common/types"

	"github.com/rs/zerolog/log"
)

type Repository struct {
	Mongo      *mongoRepository.MongoRepository
	Redis      *redisRepository.Redis
	ConnectRPC *ConnectRPCService
}

func NewRepository(
	mongoCfg *types.MongoConfig,
	redisCfg *types.RedisConfig,
	businessUrl string,
) *Repository {
	mongoRepo, err := mongoRepository.NewMongoRepository(mongoCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to MongoDB")
	}

	redis, err := redisRepository.NewRedis(redisCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to Redis")
	}

	connectRPC := NewConnectRPCService(businessUrl)

	return &Repository{
		Mongo:      mongoRepo,
		ConnectRPC: connectRPC,
		Redis:      redis,
	}
}

func (r *Repository) Close(ctx context.Context) error {
	if err := r.Redis.Close(); err != nil {
		return err
	}
	return r.Mongo.Disconnect(ctx)
}
