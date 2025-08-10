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

	"ssuspy-creator-bot/config"
	"ssuspy-creator-bot/prom"
	"ssuspy-creator-bot/repository"
	"ssuspy-creator-bot/telegram"
	"ssuspy-creator-bot/telegram/locales"
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

	repo := repository.NewRepository(cfg.Mongo, cfg.Redis, cfg.BusinessURL)
	defer repo.Close(ctx)

	go func() {
		if err := telegram.RunTelegram(ctx, &cfg, repo); err != nil {
			log.Fatal().Err(err).Msg("bot web server down")
		}
	}()

	go func() {
		if err := prom.RunMetrics(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("web server down")
		}
	}()

	quit := make(chan bool)
	<-quit
}
