package telegram

import (
	"context"
	"net/http"
	"ssuspy-bot/config"
	"ssuspy-bot/grpc_server"
	"ssuspy-bot/redis"
	"ssuspy-bot/repository"
	"ssuspy-bot/telegram/files"
	"ssuspy-bot/telegram/manager"

	"github.com/rs/zerolog/log"
)

const (
	BUSINESS_URL = "http://business-bot:8080"
)

func RunTelegram(ctx context.Context, mux *http.ServeMux, mongo *repository.MongoRepository, rdb *redis.Redis) {
	mng := manager.NewBotManager(mongo, rdb, mux, BUSINESS_URL)

	bots, err := mongo.AllBots(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load bots from database")
	} else {
		for _, bot := range bots {
			err := mng.AddBot(ctx, bot.ID, bot.SecretToken)
			if err != nil {
				log.Error().Err(err).Int64("botID", bot.ID).Msg("Failed to start bot from database")
			}
		}
	}

	filesWorker := files.NewWorker(mongo, rdb, mng)
	for i := range config.Config.FilesWorkers {
		go filesWorker.Work(ctx)
		log.Info().Int("workerID", i+1).Msg("Worker started")
	}

	go func() {
		if err := grpc_server.StartGRPCServer("50051", mng, mongo); err != nil {
			log.Fatal().Err(err).Msg("Failed to start gRPC server")
		}
	}()
}
