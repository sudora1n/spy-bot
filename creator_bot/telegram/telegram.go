package telegram

import (
	"context"
	"net/http"
	"ssuspy-common/repository/redisRepository"
	commonMiddleware "ssuspy-common/telegram/middlewares"
	"ssuspy-creator-bot/config"
	"ssuspy-creator-bot/consts"
	"ssuspy-creator-bot/repository"
	"ssuspy-creator-bot/telegram/handlers"
	"ssuspy-creator-bot/telegram/middleware"
	"ssuspy-creator-bot/telegram/utils"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/rs/zerolog/log"
)

const (
	WEBHOOK_URL = "http://creator-bot:8080/bot"
)

func RunTelegram(
	ctx context.Context,
	cfg *config.StructConfig,
	repo *repository.Repository,
) error {
	mux := http.NewServeMux()

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

	var updates <-chan telego.Update
	updates, err = bot.UpdatesViaWebhook(
		ctx,
		telego.WebhookHTTPServeMux(mux, "POST /bot", bot.SecretToken()),
		telego.WithWebhookBuffer(128),
		telego.WithWebhookSet(ctx, &telego.SetWebhookParams{
			URL:         WEBHOOK_URL,
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
	log.Info().Str("localBotApiURL", cfg.TelegramBot.ApiURL).Str("url", WEBHOOK_URL).Msg("successfully set webhook (local bot api)")

	bh, err := th.NewBotHandler(bot, updates)
	if err != nil {
		log.Fatal().Err(err).Msg("error create bot handlers") // useless xd
	}

	defer func() { _ = bh.Stop() }()

	bh.Use(th.PanicRecoveryHandler(middleware.LogPanicHandler))
	bh.Use(commonMiddleware.AutoRespond)

	middlewareGroup := middleware.NewMiddlewareGroup(repo)
	bh.Use(middlewareGroup.GetInternalUserMiddleware)

	handlerGroup := handlers.NewHandlerGroup(repo)
	bh.Handle(utils.WithProm("handleBlocked", handlerGroup.HandleBlocked), th.AnyMyChatMember())

	{
		starndard := bh.Group(th.Or(
			th.And(
				th.AnyCallbackQueryWithMessage(),
				th.CallbackDataPrefix("+"),
			),
			th.AnyMessageWithText(),
		))
		starndard.Use(middlewareGroup.RateLimitMiddleware(&redisRepository.RateLimitConfig{
			Window:    10 * time.Second,
			Limit:     5,
			QueueSize: 3,

			COUNT_KEY: consts.REDIS_RATELIMIT_COUNT,
			QUEUE_KEY: consts.REDIS_RATELIMIT_QUEUE,
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

	if err := bh.Start(); err != nil {
		log.Fatal().Err(err).Msg("bot error while process")
	}

	return http.ListenAndServe(":8080", mux)
}
