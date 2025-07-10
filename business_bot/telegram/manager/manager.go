package manager

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"ssuspy-bot/config"
	"ssuspy-bot/consts"
	"ssuspy-bot/redis"
	"ssuspy-bot/repository"
	"ssuspy-bot/telegram/handlers"
	"ssuspy-bot/telegram/middleware"
	"ssuspy-bot/telegram/utils"
	"sync"
	"time"

	"github.com/mymmrac/telego"
	ta "github.com/mymmrac/telego/telegoapi"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/rs/zerolog/log"
)

type BotInstance struct {
	ID      int64
	Bot     *telego.Bot
	Handler *th.BotHandler
	Updates <-chan telego.Update
	Cancel  context.CancelFunc
	Running bool
}

type BotManager struct {
	service *repository.MongoRepository
	rdb     *redis.Redis

	bots    map[int64]*BotInstance
	mutex   sync.RWMutex
	mux     *http.ServeMux
	baseURL string
}

func NewBotManager(service *repository.MongoRepository, rdb *redis.Redis, mux *http.ServeMux, baseURL string) *BotManager {
	return &BotManager{
		bots:    make(map[int64]*BotInstance),
		mux:     mux,
		baseURL: baseURL,
		service: service,
		rdb:     rdb,
	}
}

func (b *BotManager) AddBot(ctx context.Context, botID int64, token string) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if _, exists := b.bots[botID]; exists {
		return fmt.Errorf("bot with ID %d already exists", botID)
	}

	bot, err := telego.NewBot(
		token,
		telego.WithAPICaller(&ta.RetryCaller{
			Caller:       ta.DefaultFastHTTPCaller,
			MaxAttempts:  4,
			ExponentBase: 2,
			StartDelay:   time.Millisecond * 10,
			MaxDelay:     time.Second,
		}),
		telego.WithAPIServer(config.Config.TelegramBot.ApiURL),
	)
	if err != nil {
		return fmt.Errorf("failed to create bot: %w", err)
	}

	botUser, err := bot.GetMe(ctx)
	if err != nil {
		return fmt.Errorf("error when executing GetMe request: %w", err)
	}
	if !botUser.CanConnectToBusiness {
		return fmt.Errorf("cannot use this bot: please enable business connections in @botfather")
	}
	// не ставлю !botUser.SupportsInlineQueries для поддержки старых ботов

	commands := []telego.BotCommand{
		{
			Command:     "start",
			Description: "main menu",
		},
	}

	if config.Config.BusinessGithubURL != "" {
		commands = append(commands, telego.BotCommand{
			Command:     "github",
			Description: "bot source code",
		})
	}

	err = bot.SetMyCommands(ctx, &telego.SetMyCommandsParams{
		Commands: commands,
	})
	if err != nil {
		log.Warn().Int64("botID", botID).Err(err).Msg("failed set bot commands")
	}

	webhookURL := fmt.Sprintf("%s/bot_%d", b.baseURL, botID)
	webhookPath := fmt.Sprintf("POST /bot_%d", botID)

	botCtx, botCancel := context.WithCancel(context.Background())

	updates, err := bot.UpdatesViaWebhook(
		botCtx,
		telego.WebhookHTTPServeMux(b.mux, webhookPath, bot.SecretToken()),
		telego.WithWebhookBuffer(128),
		telego.WithWebhookSet(ctx, &telego.SetWebhookParams{
			URL:         webhookURL,
			SecretToken: bot.SecretToken(),
			AllowedUpdates: []string{
				"update_id",
				"message",
				"business_connection",
				"business_message",
				"edited_business_message",
				"deleted_business_messages",
				"my_chat_member",
				"callback_query",
				"inline_query",
				"chosen_inline_result",
			},
			DropPendingUpdates: false,
		}),
	)
	if err != nil {
		botCancel()
		return fmt.Errorf("failed to setup webhook updates: %w", err)
	}

	botHandler, err := th.NewBotHandler(bot, updates)
	if err != nil {
		botCancel()
		return fmt.Errorf("failed to create bot handler: %w", err)
	}

	instance := &BotInstance{
		ID:      botID,
		Bot:     bot,
		Handler: botHandler,
		Updates: updates,
		Cancel:  botCancel,
		Running: true,
	}

	b.setupBotHandlers(instance)

	go func() {
		defer func() {
			instance.Cancel()
		}()

		if err := botHandler.Start(); err != nil {
			log.Error().Err(err).Int64("botID", botID).Msg("bot handler stopped with error")
		}
	}()

	b.bots[botID] = instance

	log.Debug().Int64("botID", botID).Str("webhookURL", webhookURL).Msg("bot started successfully")
	return nil
}

func (b *BotManager) RemoveBot(botID int64) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	instance, exists := b.bots[botID]
	if !exists {
		return fmt.Errorf("bot with ID %d not found", botID)
	}

	if instance.Cancel != nil {
		instance.Cancel()
	}
	instance.Handler.Stop()
	instance.Running = false

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := instance.Bot.DeleteWebhook(ctx, &telego.DeleteWebhookParams{
		DropPendingUpdates: false,
	})
	if err != nil {
		log.Warn().Err(err).Int64("botID", botID).Msg("failed to delete webhook")
	}

	delete(b.bots, botID)

	log.Info().Int64("botID", botID).Msg("bot stopped and removed successfully")
	return nil
}

func (b *BotManager) GetBot(botID int64) (*BotInstance, bool) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	bot, exists := b.bots[botID]
	return bot, exists
}

func (b *BotManager) ListBots() []int64 {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	var botIDs []int64
	for id := range b.bots {
		botIDs = append(botIDs, id)
	}
	return botIDs
}

func (b *BotManager) setupBotHandlers(instance *BotInstance) {
	instance.Handler.Use(th.PanicRecoveryHandler(middleware.LogPanicHandler))

	middlewareGroup := middleware.NewMiddlewareGroup(b.service, b.rdb)
	instance.Handler.Use(middlewareGroup.BotContextMiddleware(instance.ID))
	instance.Handler.Use(middleware.SkipNonPrivateChatsMiddleware)
	instance.Handler.Use(middlewareGroup.GetInternalUserMiddleware)

	handlerGroup := handlers.NewHandlerGroup(b.service, b.rdb)
	instance.Handler.Handle(utils.WithProm("handleBlocked", handlerGroup.HandleBlocked), th.AnyMyChatMember())

	{
		inline := instance.Handler.Group(th.AnyInlineQuery())
		// inline.Use(middlewareGroup.RateLimitMiddleware(middleware.RateLimitConfig{
		// 	Window: 5 * time.Second,
		// 	Limit:  10,
		// }))
		inline.Use(middlewareGroup.SyncUserMiddleware)
		inline.Handle(
			utils.WithProm("handleInlineQuery", handlers.HandleInlineQuery),
			th.AnyInlineQuery(),
		)
	}

	{
		chosenInline := instance.Handler.Group(th.AnyChosenInlineResult())
		// chosenInline.Use(middlewareGroup.RateLimitMiddleware(middleware.RateLimitConfig{
		// 	Window: 10 * time.Second,
		// 	Limit:  5,
		// }))
		chosenInline.Use(middlewareGroup.SyncUserMiddleware)
		chosenInline.Handle(
			utils.WithProm(
				"handleUserGiftUpgrade",
				handlers.HandleUserGiftUpgrade,
			),
			th.AnyChosenInlineResult(),
		)
	}

	{
		standard := instance.Handler.Group(th.Or(
			th.And(
				th.AnyCallbackQueryWithMessage(),
				th.CallbackDataPrefix("_"),
			),
			th.AnyCommand(),
		))
		standard.Use(middlewareGroup.RateLimitMiddleware(middleware.RateLimitConfig{
			Window:    10 * time.Second,
			Limit:     5,
			QueueSize: 3,
		}))
		standard.Use(middlewareGroup.SyncUserMiddleware)
		standard.Handle(
			utils.WithProm("handleStart", handlers.HandleStart),
			th.Or(
				th.CallbackDataEqual(consts.CALLBACK_PREFIX_BACK_TO_START),
				th.CommandEqual("start"),
			),
		)
		if config.Config.BusinessGithubURL != "" {
			standard.Handle(utils.WithProm("handleGithub", handlers.HandleGithub), th.CommandEqual("github"))
		}
		standard.Handle(
			utils.WithProm("handleSettings", handlerGroup.HandleSettings),
			th.CallbackDataPrefix(consts.CALLBACK_PREFIX_SETTINGS),
			th.AnyCallbackQueryWithMessage(),
		)
		standard.Handle(
			utils.WithProm("handleSettingsDeleted", handlerGroup.HandleSettingsDeleted),
			th.CallbackDataPrefix(consts.CALLBACK_PREFIX_SETTINGS_DELETED),
			th.AnyCallbackQueryWithMessage(),
		)
		standard.Handle(
			utils.WithProm("handleSettingsEdited", handlerGroup.HandleSettingsEdited),
			th.CallbackDataPrefix(consts.CALLBACK_PREFIX_SETTINGS_EDITED),
			th.AnyCallbackQueryWithMessage(),
		)
		standard.Handle(
			utils.WithProm("handleLanguage", handlers.HandleLanguage),
			th.CallbackDataEqual(consts.CALLBACK_PREFIX_LANG),
			th.AnyCallbackQueryWithMessage(),
		)
		standard.Handle(
			utils.WithProm("handleLanguageChange", handlerGroup.HandleLanguageChange),
			th.CallbackDataPrefix(consts.CALLBACK_PREFIX_LANG_CHANGE),
			th.AnyCallbackQueryWithMessage(),
		)

		standard.Handle(
			utils.WithProm("handleDeletedLog", handlerGroup.HandleDeletedLog),
			th.CallbackDataPrefix(consts.CALLBACK_PREFIX_DELETED_LOG),
			th.AnyCallbackQueryWithMessage(),
		)
		standard.Handle(
			utils.WithProm("handleDeletedMessage", handlerGroup.HandleDeletedMessage),
			th.CallbackDataPrefix(consts.CALLBACK_PREFIX_DELETED_MESSAGE),
			th.AnyCallbackQueryWithMessage(),
		)
		standard.Handle(
			utils.WithProm("handleDeletedMessageDetails", handlerGroup.HandleDeletedMessageDetails),
			th.CallbackDataPrefix(consts.CALLBACK_PREFIX_DELETED_DETAILS),
			th.AnyCallbackQueryWithMessage(),
		)
		standard.Handle(
			utils.WithProm("handleGetDeletedFiles", handlerGroup.HandleGetDeletedFiles),
			th.CallbackDataPrefix(consts.CALLBACK_PREFIX_DELETED_FILES),
			th.AnyCallbackQueryWithMessage(),
		)

		edited := standard.Group(th.AnyCallbackQueryWithMessage())
		edited.Use(middlewareGroup.EditedGetMessages)
		edited.Handle(
			utils.WithProm("handleEditedLog", handlerGroup.HandleEditedLog),
			th.CallbackDataPrefix(consts.CALLBACK_PREFIX_EDITED_LOG),
			th.AnyCallbackQueryWithMessage(),
		)
		edited.Handle(
			utils.WithProm("handleEditedFiles", handlerGroup.HandleEditedFiles),
			th.CallbackDataPrefix(consts.CALLBACK_PREFIX_EDITED_FILES),
			th.AnyCallbackQueryWithMessage(),
		)
	}

	{
		businessConnection := instance.Handler.Group(th.AnyBusinessConnection())
		businessConnection.Use(middlewareGroup.IsolationMiddleware(consts.REDIS_RATELIMIT_QUEUE_BUSINESS_CONNECTION, 5))
		businessConnection.Use(middlewareGroup.SyncUserMiddleware)
		businessConnection.Handle(
			utils.WithProm("handleConnection", handlerGroup.HandleConnection),
			th.AnyBusinessConnection(),
		)
	}

	{
		business := instance.Handler.Group(th.Or(
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
		business.Handle(
			utils.WithProm("handleEdited", handlerGroup.HandleEdited),
			th.AnyEditedBusinessMessage(),
		)
	}

	{
		businessMessage := instance.Handler.Group(th.AnyBusinessMessage())
		businessMessage.Use(middlewareGroup.BusinessGetUserMiddleware)

		startsWith := regexp.MustCompile(`^\..*`)
		userCommands := businessMessage.Group(th.AnyBusinessMessage(), utils.BusinessMessageMatches(startsWith))
		userCommands.Use(middlewareGroup.RateLimitMiddleware(middleware.RateLimitConfig{
			Window:    10 * time.Second,
			Limit:     3,
			QueueSize: 1,
		}))
		userCommands.Use(middlewareGroup.BusinessIsFromUser)
		userCommands.Use(middlewareGroup.BusinessIgnoreMessage)
		userCommands.Use(middlewareGroup.BusinessUserSetRights)

		helpRegex := regexp.MustCompile(`^\s*\.help\b`)
		userCommands.Handle(
			utils.WithProm("handleUserHelp", handlerGroup.HandleUserHelp),
			utils.BusinessMessageMatches(helpRegex),
			th.AnyBusinessMessage(),
		)

		// вторая "а" - кириллическая
		animRegex := regexp.MustCompile(`^\s*\.(a|а|anim)\s`)
		userCommands.Handle(
			utils.WithProm("handleUserAnimation", handlerGroup.HandleUserAnimation),
			utils.BusinessMessageMatches(animRegex),
			th.AnyBusinessMessage(),
		)

		loveRegex := regexp.MustCompile(`^\s*\.love(ua|ru)?\b`)
		userCommands.Handle(
			utils.WithProm("handleUserLove", handlerGroup.HandleUserLove),
			utils.BusinessMessageMatches(loveRegex),
			th.AnyBusinessMessage(),
		)

		businessMessage.Handle(utils.WithProm("handleMessage", handlerGroup.HandleMessage), th.AnyBusinessMessage())
	}
}
