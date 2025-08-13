package repository

import (
	"context"
	"ssuspy-common/repository/mongoRepository"
	"ssuspy-common/types"

	"github.com/rs/zerolog/log"
)

type Repository struct {
	Mongo *mongoRepository.MongoRepository
}

func NewRepository(
	mongoCfg *types.MongoConfig,
) *Repository {
	mongoRepo, err := mongoRepository.NewMongoRepository(mongoCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to MongoDB")
	}

	return &Repository{
		Mongo: mongoRepo,
	}
}

func (r *Repository) Close(ctx context.Context) error {
	return r.Mongo.Disconnect(ctx)
}
