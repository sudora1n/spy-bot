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
	"ssuspy-bot/connectrpc"
	"ssuspy-bot/metrics"
	"ssuspy-bot/repository"
	"ssuspy-bot/telegram"
	"ssuspy-bot/telegram/locales"
	"ssuspy-bot/telegram/manager"
)

const (
	BUSINESS_URL = "http://business-bot"

	METRICS_PORT    = 8080
	TELEGRAM_PORT   = 8081
	CONNECTRPC_PORT = 8082
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

	repo := repository.NewRepository(cfg.Mongo, cfg.Redis)
	defer repo.Close(ctx)

	go func() {
		if err := metrics.RunMetrics(METRICS_PORT); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("web server of metrics is down")
		}
	}()

	telegramUrl := fmt.Sprintf("%s:%d", BUSINESS_URL, TELEGRAM_PORT)
	telegramMux := http.NewServeMux()
	telegramManager := manager.NewBotManager(repo, telegramMux, telegramUrl)

	telegram.InitTelegram(ctx, repo, telegramManager)
	go func() {
		if err := telegram.RunTelegram(telegramMux, TELEGRAM_PORT); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("web server of telegram is down")
		}
	}()

	go func() {
		if err := connectrpc.RunConnectRPC(telegramManager, repo, CONNECTRPC_PORT); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("web server of telegram is down")
		}
	}()

	quit := make(chan bool)
	<-quit
}
