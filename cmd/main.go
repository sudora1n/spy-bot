package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/ziflex/lecho/v3"
	"golang.org/x/text/language"

	"ssuspy-bot/config"
	"ssuspy-bot/consts"
	"ssuspy-bot/files"
	"ssuspy-bot/handlers"
	"ssuspy-bot/locales"
	"ssuspy-bot/middleware"
	"ssuspy-bot/redis"
	"ssuspy-bot/repository"
	"ssuspy-bot/utils"
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

	options := []telego.BotOption{}
	if cfg.TelegramBot.ApiURL != "" {
		options = append(options, telego.WithAPIServer(cfg.TelegramBot.ApiURL))
	}

	bot, err := telego.NewBot(cfg.TelegramBot.Token, options...)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create bot")
	}

	botUser, err := bot.GetMe(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("error when executing GetMe request")
	}
	if !botUser.CanConnectToBusiness {
		log.Fatal().Msg("cannot use this bot: please enable business connections in @botfather")
	}

	commands := []telego.BotCommand{
		{
			Command:     "start",
			Description: "main menu",
		},
	}

	hasGithub := cfg.GithubURL != ""
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

	updates, err := bot.UpdatesViaLongPolling(ctx, &telego.GetUpdatesParams{
		Timeout: 10,
		AllowedUpdates: []string{
			"update_id",
			"message",
			"business_connection",
			"business_message",
			"edited_business_message",
			"deleted_business_messages",
			"my_chat_member",
			"callback_query",
		},
	})
	if err != nil {
		log.Fatal().Err(err).Msg("error create bot updates")
	}

	bh, err := th.NewBotHandler(bot, updates)
	if err != nil {
		log.Fatal().Err(err).Msg("error create bot handlers") // useless xd
	}

	defer func() { _ = bh.Stop() }()

	bh.Use(th.PanicRecoveryHandler(middleware.LogPanicHandler))

	middlewareGroup := middleware.NewMiddlewareGroup(mongoRepo, &rdb)
	bh.Use(middlewareGroup.GetInternalUserMiddleware)

	handlerGroup := handlers.NewHandlerGroup(mongoRepo, &rdb)
	bh.Handle(utils.WithProm("handleBlocked", handlerGroup.HandleBlocked), th.AnyMyChatMember())

	{
		starndard := bh.Group(th.Or(
			th.And(
				th.AnyCallbackQueryWithMessage(),
				th.CallbackDataPrefix("_"),
			),
			th.AnyCommand(),
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
			starndard.Handle(utils.WithProm("handleGithub", handlers.HandleGithub), th.CommandEqual("github"), th.AnyMessage())
		}
		starndard.Handle(utils.WithProm("handleLanguage", handlers.HandleLanguage), th.CallbackDataEqual(consts.CALLBACK_PREFIX_LANG), th.AnyCallbackQueryWithMessage())
		starndard.Handle(
			utils.WithProm("handleLanguageChange", handlerGroup.HandleLanguageChange),
			th.CallbackDataPrefix(consts.CALLBACK_PREFIX_LANG_CHANGE),
			th.AnyCallbackQueryWithMessage(),
		)

		starndard.Handle(utils.WithProm("handleDeletedLog", handlerGroup.HandleDeletedLog), th.CallbackDataPrefix(consts.CALLBACK_PREFIX_DELETED_LOG), th.AnyCallbackQueryWithMessage())
		starndard.Handle(utils.WithProm("handleDeletedMessage", handlerGroup.HandleDeletedMessage), th.CallbackDataPrefix(consts.CALLBACK_PREFIX_DELETED_MESSAGE), th.AnyCallbackQueryWithMessage())
		starndard.Handle(utils.WithProm("handleDeletedMessageDetails", handlerGroup.HandleDeletedMessageDetails), th.CallbackDataPrefix(consts.CALLBACK_PREFIX_DELETED_DETAILS), th.AnyCallbackQueryWithMessage())
		starndard.Handle(utils.WithProm("handleGetDeletedFiles", handlerGroup.HandleGetDeletedFiles), th.CallbackDataPrefix(consts.CALLBACK_PREFIX_DELETED_FILES), th.AnyCallbackQueryWithMessage())
		edited := starndard.Group(th.AnyCallbackQueryWithMessage())
		edited.Use(middlewareGroup.EditedGetMessages)
		edited.Handle(utils.WithProm("handleEditedLog", handlerGroup.HandleEditedLog), th.CallbackDataPrefix(consts.CALLBACK_PREFIX_EDITED_LOG), th.AnyCallbackQueryWithMessage())
		edited.Handle(utils.WithProm("handleEditedFiles", handlerGroup.HandleEditedFiles), th.CallbackDataPrefix(consts.CALLBACK_PREFIX_EDITED_FILES), th.AnyCallbackQueryWithMessage())
	}

	{
		businessConnection := bh.Group(th.AnyBusinessConnection())
		businessConnection.Use(middlewareGroup.IsolationMiddleware(consts.REDIS_RATELIMIT_QUEUE_BUSINESS_CONNECTION, 5))
		businessConnection.Use(middlewareGroup.SyncUserMiddleware)
		businessConnection.Handle(utils.WithProm("handleConnection", handlerGroup.HandleConnection), th.AnyBusinessConnection())
	}

	{
		business := bh.Group(th.Or(
			th.AnyDeletedBusinessMessages(),
			th.AnyEditedBusinessMessage(),
			th.And(
				th.AnyCallbackQueryWithMessage(),
				th.CallbackDataPrefix("-"),
			),
		))
		business.Use(middlewareGroup.IsolationMiddleware(consts.REDIS_RATELIMIT_QUEUE_BUSINESS, 20))
		business.Use(middlewareGroup.BusinessGetUserMiddleware)
		business.Handle(
			utils.WithProm("handleDeleted", handlerGroup.HandleDeleted),
			th.Or(
				th.CallbackDataPrefix(consts.CALLBACK_PREFIX_DELETED),
				th.AnyDeletedBusinessMessages(),
			))
		business.Handle(utils.WithProm("handleEdited", handlerGroup.HandleEdited), th.AnyEditedBusinessMessage())
	}

	{
		businessMessage := bh.Group(th.AnyBusinessMessage())
		businessMessage.Use(middlewareGroup.BusinessGetUserMiddleware)
		businessMessage.Handle(utils.WithProm("handleMessage", handlerGroup.HandleMessage), th.AnyBusinessMessage())
	}

	filesWorker := files.NewWorker(mongoRepo, &rdb, bot)
	for i := range cfg.FilesWorkers {
		go filesWorker.Work(ctx, i+1)
	}

	e := echo.New()
	e.Logger = lecho.From(log.Logger)

	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
	go func() {
		if err := e.Start(":8080"); err != nil {
			log.Fatal().Err(err).Msg("echo/labstack is down")
		}
	}()

	if err = bh.Start(); err != nil {
		log.Fatal().Err(err).Msg("bot error while process")
	}
}
