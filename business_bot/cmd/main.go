package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/text/language"

	"ssuspy-bot/config"
	"ssuspy-bot/files"
	"ssuspy-bot/grpc_server"
	"ssuspy-bot/locales"
	"ssuspy-bot/manager"
	"ssuspy-bot/redis"
	"ssuspy-bot/repository"
)

func main() {
	ctx := context.Background()

	zerolog.TimeFieldFormat = time.RFC3339
	// zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	var logLvl zerolog.Level
	if cfg.DevMode {
		logLvl = zerolog.DebugLevel
	} else {
		logLvl = zerolog.InfoLevel
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out: os.Stderr,
		FormatCaller: func(i any) string {
			_, file := filepath.Split(fmt.Sprintf("%v", i))
			return file
		},
	}).With().Timestamp().Caller().Logger().Level(logLvl)

	if err := locales.Init(language.English); err != nil {
		log.Fatal().Err(err).Msg("failed to initialize i18n")
	}

	mongoRepo, err := repository.NewMongoRepository(cfg.Mongo)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to MongoDB")
	}
	defer mongoRepo.Disconnect(ctx)

	rdb, err := redis.NewRedis(cfg.Redis)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to Redis")
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	go func() {
		if err := http.ListenAndServe(":8080", mux); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("web server server down")
		}
	}()

	url := "http://business-bot:8080"
	mng := manager.NewBotManager(mongoRepo, &rdb, mux, url)

	bots, err := mongoRepo.AllBots(ctx)
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

	filesWorker := files.NewWorker(mongoRepo, &rdb, mng)
	for i := range cfg.FilesWorkers {
		go filesWorker.Work(ctx)
		log.Info().Int("workerID", i+1).Msg("Worker started")
	}

	go func() {
		if err := grpc_server.StartGRPCServer("50051", mng, mongoRepo); err != nil {
			log.Fatal().Err(err).Msg("Failed to start gRPC server")
		}
	}()

	quit := make(chan bool)
	<-quit
}
