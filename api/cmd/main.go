package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"ssuspy-api/config"
	"ssuspy-api/repository"
	"ssuspy-api/web"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type AuthType string

const (
	AuthTypeJWT      AuthType = "jwt"
	AuthTypeInternal AuthType = "internal"
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

	repo := repository.NewRepository(cfg.Mongo)
	defer repo.Close(ctx)

	err = web.RunWeb(repo, &cfg, 3000)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal().Err(err).Msg("web server is down")
	}
}
