package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/text/language"

	"ssuspy-bot/config"
	"ssuspy-bot/metrics"
	"ssuspy-bot/redis"
	"ssuspy-bot/repository"
	"ssuspy-bot/telegram"
	"ssuspy-bot/telegram/locales"
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
	go metrics.RunMetrics(mux)

	telegram.RunTelegram(ctx, mux, mongoRepo, &rdb)

	quit := make(chan bool)
	<-quit
}
