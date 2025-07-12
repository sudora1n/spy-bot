package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/text/language"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"ssuspy-creator-bot/config"
	"ssuspy-creator-bot/consts"
	"ssuspy-creator-bot/redis"
	"ssuspy-creator-bot/repository"
	"ssuspy-creator-bot/telegram/handlers"
	"ssuspy-creator-bot/telegram/locales"
	"ssuspy-creator-bot/telegram/middleware"
	"ssuspy-creator-bot/telegram/utils"
	managerv1 "ssuspy-proto/gen/manager/v1"
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

	grpcConn, err := grpc.NewClient(
		fmt.Sprint(cfg.Grpc),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to GRPC")
	}
	grpcClient := managerv1.NewManagerServiceClient(grpcConn)

	bot, err := telego.NewBot(cfg.TelegramBot.Token, telego.WithAPIServer(cfg.TelegramBot.ApiURL))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create bot")
	}

	botUser, err := bot.GetMe(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("error when executing GetMe request")
	}

	commands := []telego.BotCommand{
		{
			Command:     "start",
			Description: "main menu",
		},
	}

	hasGithub := cfg.CreatorGithubURL != ""
	if hasGithub {
		commands = append(commands, telego.BotCommand{
			Command:     "github",
			Description: "bot source code",
		})
	}

	bot.SetMyCommands(ctx, &telego.SetMyCommandsParams{
		Commands: commands,
	})

	log.Info().Msgf("bot username: @%s", botUser.Username)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	var updates <-chan telego.Update
	url := "http://creator-bot:8080/bot"
	updates, err = bot.UpdatesViaWebhook(
		ctx,
		telego.WebhookHTTPServeMux(mux, "POST /bot", bot.SecretToken()),
		telego.WithWebhookBuffer(128),
		telego.WithWebhookSet(ctx, &telego.SetWebhookParams{
			URL:         url,
			SecretToken: bot.SecretToken(),
			AllowedUpdates: []string{
				"update_id",
				"message",
				"callback_query",
				"my_chat_member",
			},
			DropPendingUpdates: false,
		}),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("error create bot updates via webhook (local bot api)")
	}
	log.Info().Str("localBotApiURL", cfg.TelegramBot.ApiURL).Str("url", url).Msg("successfully set webhook (local bot api)")

	bh, err := th.NewBotHandler(bot, updates)
	if err != nil {
		log.Fatal().Err(err).Msg("error create bot handlers") // useless xd
	}

	defer func() { _ = bh.Stop() }()

	bh.Use(th.PanicRecoveryHandler(middleware.LogPanicHandler))
	bh.Use(middleware.AutoRespond)

	middlewareGroup := middleware.NewMiddlewareGroup(mongoRepo, &rdb)
	bh.Use(middlewareGroup.GetInternalUserMiddleware)

	handlerGroup := handlers.NewHandlerGroup(mongoRepo, grpcClient)
	bh.Handle(utils.WithProm("handleBlocked", handlerGroup.HandleBlocked), th.AnyMyChatMember())

	{
		starndard := bh.Group(th.Or(
			th.And(
				th.AnyCallbackQueryWithMessage(),
				th.CallbackDataPrefix("+"),
			),
			th.AnyMessageWithText(),
		))
		starndard.Use(middlewareGroup.RateLimitMiddleware(middleware.RateLimitConfig{
			Window:    10 * time.Second,
			Limit:     5,
			QueueSize: 3,
		}))
		starndard.Use(middlewareGroup.SyncUserMiddleware)
		starndard.Handle(utils.WithProm("handleStart", handlers.HandleStart), th.Or(
			th.CallbackDataEqual(consts.CALLBACK_PREFIX_BACK_TO_START),
			th.CommandEqual("start"),
		))
		if hasGithub {
			starndard.Handle(utils.WithProm("handleGithub", handlers.HandleGithub), th.CommandEqual("github"))
		}
		starndard.Handle(utils.WithProm("handleLanguage", handlers.HandleLanguage), th.CallbackDataEqual(consts.CALLBACK_PREFIX_LANG), th.AnyCallbackQueryWithMessage())
		starndard.Handle(
			utils.WithProm("handleLanguageChange", handlerGroup.HandleLanguageChange),
			th.CallbackDataPrefix(consts.CALLBACK_PREFIX_LANG_CHANGE),
			th.AnyCallbackQueryWithMessage(),
		)
		starndard.Handle(utils.WithProm("handleBotsList", handlerGroup.HandleBotsList), th.CallbackDataPrefix(consts.CALLBACK_PREFIX_BOT_LIST), th.AnyCallbackQueryWithMessage())
		starndard.Handle(utils.WithProm("handleBotItem", handlerGroup.HandleBotItem), th.CallbackDataPrefix(consts.CALLBACK_PREFIX_BOT_ITEM), th.AnyCallbackQueryWithMessage())
		starndard.Handle(utils.WithProm("handleBotRemove", handlerGroup.HandleBotRemove), th.CallbackDataPrefix(consts.CALLBACK_PREFIX_BOT_REMOVE), th.AnyCallbackQueryWithMessage())
		starndard.Handle(utils.WithProm("handleToken", handlerGroup.HandleToken), th.AnyMessageWithText())
	}

	go func() {
		if err := bh.Start(); err != nil {
			log.Fatal().Err(err).Msg("bot error while process")
		}
	}()

	go func() {
		if err := http.ListenAndServe(":8080", mux); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("web server down")
		}
	}()

	quit := make(chan bool)
	<-quit
}
