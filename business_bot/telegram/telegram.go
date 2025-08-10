package telegram

import (
	"context"
	"ssuspy-bot/config"
	"ssuspy-bot/repository"
	"ssuspy-bot/telegram/files"
	"ssuspy-bot/telegram/manager"

	"github.com/rs/zerolog/log"
)

func InitTelegram(
	ctx context.Context,
	repository *repository.Repository,
	mng *manager.BotManager,
) {
	bots, err := repository.Mongo.AllBots(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load bots from database")
	} else {
		for _, bot := range bots {
			err := mng.AddBot(ctx, bot.Id, bot.SecretToken)
			if err != nil {
				log.Error().Err(err).Int64("botID", bot.Id).Msg("Failed to start bot from database")
			}
		}
	}

	filesWorker := files.NewWorker(repository, mng)
	for i := range config.Config.FilesWorkers {
		go filesWorker.Work(ctx)
		log.Info().Int("workerID", i+1).Msg("Worker started")
	}
}
