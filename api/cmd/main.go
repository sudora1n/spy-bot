package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"ssuspy-api/api"
	"ssuspy-api/config"
	"ssuspy-api/repository"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"ssuspy-proto/gen/bots/v1/botsv1connect"
	"ssuspy-proto/gen/manager/v1/managerv1connect"
	"ssuspy-proto/gen/messages/v1/messagesv1connect"
	"ssuspy-proto/gen/users/v1/usersv1connect"
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

	mongoRepo, err := repository.NewMongoRepository(cfg.Mongo)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to MongoDB")
	}
	defer mongoRepo.Disconnect(ctx)

	business := managerv1connect.NewManagerServiceClient(
		http.DefaultClient,
		cfg.BusinessURL,
	)

	r := chi.NewRouter()

	botsServer := api.NewBotsServer(mongoRepo, business)
	path, handler := botsv1connect.NewBotsServiceHandler(
		botsServer,
	)
	r.Mount(path, h2c.NewHandler(handler, &http2.Server{}))

	usersServer := &api.UsersServer{}
	path, handler = usersv1connect.NewUsersServiceHandler(
		usersServer,
	)
	r.Mount(path, h2c.NewHandler(handler, &http2.Server{}))

	messagesServer := &api.MessagesServer{}
	path, handler = messagesv1connect.NewMessagesServiceHandler(
		messagesServer,
	)
	r.Mount(path, h2c.NewHandler(handler, &http2.Server{}))

	http.ListenAndServe(":3000", r)
}
