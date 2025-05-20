package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/text/language"

	bundlei18n "ssuspy-bot/bundle_i18n"
	"ssuspy-bot/config"
	"ssuspy-bot/consts"
	"ssuspy-bot/files"
	"ssuspy-bot/handlers"
	"ssuspy-bot/middleware"
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

	if err := bundlei18n.Init("locales", language.English); err != nil {
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
	bh.HandleMyChatMemberUpdated(handlerGroup.HandleBlocked)

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
		starndard.Handle(handlers.HandleStart, th.Or(
			th.CallbackDataEqual(consts.CALLBACK_PREFIX_BACK_TO_START),
			th.CommandEqual("start"),
		))
		if hasGithub {
			starndard.HandleMessage(handlers.HandleGithub, th.CommandEqual("github"))
		}
		starndard.HandleCallbackQuery(handlers.HandleLanguage, th.CallbackDataEqual(consts.CALLBACK_PREFIX_LANG))
		starndard.HandleCallbackQuery(handlerGroup.HandleLanguageChange, th.CallbackDataPrefix(consts.CALLBACK_PREFIX_LANG_CHANGE))

		starndard.HandleCallbackQuery(handlerGroup.HandleDeletedLog, th.CallbackDataPrefix(consts.CALLBACK_PREFIX_DELETED_LOG))
		starndard.HandleCallbackQuery(handlerGroup.HandleDeletedMessage, th.CallbackDataPrefix(consts.CALLBACK_PREFIX_DELETED_MESSAGE))
		starndard.HandleCallbackQuery(handlerGroup.HandleDeletedMessageDetails, th.CallbackDataPrefix(consts.CALLBACK_PREFIX_DELETED_DETAILS))
		starndard.HandleCallbackQuery(handlerGroup.HandleGetDeletedFiles, th.CallbackDataPrefix(consts.CALLBACK_PREFIX_DELETED_FILES))
		starndard.HandleCallbackQuery(handlerGroup.HandleDeletedPagination, th.CallbackDataPrefix(consts.CALLBACK_PREFIX_DELETED_PAGINATION))
		edited := starndard.Group(th.AnyCallbackQueryWithMessage())
		edited.Use(middlewareGroup.EditedGetMessages)
		edited.HandleCallbackQuery(handlerGroup.HandleEditedLog, th.CallbackDataPrefix(consts.CALLBACK_PREFIX_EDITED_LOG))
		edited.HandleCallbackQuery(handlerGroup.HandleEditedFiles, th.CallbackDataPrefix(consts.CALLBACK_PREFIX_EDITED_FILES))
	}

	{
		businessConnection := bh.Group(th.AnyBusinessConnection())
		businessConnection.Use(middlewareGroup.IsolationMiddleware(consts.REDIS_RATELIMIT_QUEUE_BUSINESS_CONNECTION, 5))
		businessConnection.Use(middlewareGroup.SyncUserMiddleware)
		businessConnection.HandleBusinessConnection(handlerGroup.HandleConnection, th.AnyBusinessConnection())
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
			handlerGroup.HandleDeleted,
			th.Or(
				th.CallbackDataPrefix(consts.CALLBACK_PREFIX_DELETED),
				th.AnyDeletedBusinessMessages(),
			))
		business.HandleEditedBusinessMessage(handlerGroup.HandleEdited, th.AnyEditedBusinessMessage())
	}

	{
		businessMessage := bh.Group(th.AnyBusinessMessage())
		businessMessage.Use(middlewareGroup.BusinessGetUserMiddleware)
		businessMessage.HandleBusinessMessage(handlerGroup.HandleMessage, th.AnyBusinessMessage())
	}

	filesWorker := files.NewWorker(mongoRepo, &rdb, bot)
	for i := range cfg.FilesWorkers {
		go filesWorker.Work(ctx, i+1)
	}

	if err = bh.Start(); err != nil {
		log.Fatal().Err(err).Msg("bot error while process")
	}
}
