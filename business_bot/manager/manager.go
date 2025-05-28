package manager

import (
	"context"
	"fmt"
	"net/http"
	"ssuspy-bot/config"
	"ssuspy-bot/consts"
	"ssuspy-bot/handlers"
	"ssuspy-bot/middleware"
	"ssuspy-bot/redis"
	"ssuspy-bot/repository"
	"ssuspy-bot/utils"
	"sync"
	"time"

	"github.com/mymmrac/telego"
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

func (bm *BotManager) AddBot(ctx context.Context, botID int64, token string) error {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	if _, exists := bm.bots[botID]; exists {
		return fmt.Errorf("bot with ID %d already exists", botID)
	}

	bot, err := telego.NewBot(token, telego.WithAPIServer(config.Config.TelegramBot.ApiURL))
	if err != nil {
		return fmt.Errorf("failed to create bot: %w", err)
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

	if config.Config.GithubURL != "" {
		commands = append(commands, telego.BotCommand{
			Command:     "github",
			Description: "bot source code",
		})
	}

	bot.SetMyCommands(ctx, &telego.SetMyCommandsParams{
		Commands: commands,
	})

	webhookURL := fmt.Sprintf("%s/bot_%d", bm.baseURL, botID)
	webhookPath := fmt.Sprintf("POST /bot_%d", botID)

	updates, err := bot.UpdatesViaWebhook(
		ctx,
		telego.WebhookHTTPServeMux(bm.mux, webhookPath, bot.SecretToken()),
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
			},
			DropPendingUpdates: false,
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to setup webhook updates: %w", err)
	}

	botHandler, err := th.NewBotHandler(bot, updates)
	if err != nil {
		return fmt.Errorf("failed to create bot handler: %w", err)
	}

	botCtx, cancel := context.WithCancel(ctx)

	instance := &BotInstance{
		ID:      botID,
		Bot:     bot,
		Handler: botHandler,
		Updates: updates,
		Cancel:  cancel,
		Running: true,
	}

	bm.setupBotHandlers(instance)

	go func() {
		if err := botHandler.Start(); err != nil {
			log.Error().Err(err).Int64("botID", botID).Msg("bot handler stopped with error")
		}
	}()

	go bm.processBotUpdates(botCtx, instance)

	bm.bots[botID] = instance

	log.Info().Int64("botID", botID).Str("webhookURL", webhookURL).Msg("bot started successfully")
	return nil
}

func (bm *BotManager) RemoveBot(botID int64) error {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	instance, exists := bm.bots[botID]
	if !exists {
		return fmt.Errorf("bot with ID %d not found", botID)
	}

	instance.Cancel()
	instance.Handler.Stop()
	instance.Running = false

	ctx := context.Background()
	err := instance.Bot.DeleteWebhook(ctx, &telego.DeleteWebhookParams{
		DropPendingUpdates: false,
	})
	if err != nil {
		log.Warn().Err(err).Int64("botID", botID).Msg("failed to delete webhook")
	}

	delete(bm.bots, botID)

	log.Info().Int64("botID", botID).Msg("bot stopped and removed successfully")
	return nil
}

func (bm *BotManager) GetBot(botID int64) (*BotInstance, bool) {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	bot, exists := bm.bots[botID]
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
	instance.Handler.Use(middlewareGroup.GetInternalUserMiddleware)

	handlerGroup := handlers.NewHandlerGroup(b.service, b.rdb)
	instance.Handler.Handle(utils.WithProm("handleBlocked", handlerGroup.HandleBlocked), th.AnyMyChatMember())

	{
		starndard := instance.Handler.Group(th.Or(
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
		if config.Config.GithubURL != "" {
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
		businessConnection := instance.Handler.Group(th.AnyBusinessConnection())
		businessConnection.Use(middlewareGroup.IsolationMiddleware(consts.REDIS_RATELIMIT_QUEUE_BUSINESS_CONNECTION, 5))
		businessConnection.Use(middlewareGroup.SyncUserMiddleware)
		businessConnection.Handle(utils.WithProm("handleConnection", handlerGroup.HandleConnection), th.AnyBusinessConnection())
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
		business.Handle(utils.WithProm("handleEdited", handlerGroup.HandleEdited), th.AnyEditedBusinessMessage())
	}

	{
		businessMessage := instance.Handler.Group(th.AnyBusinessMessage())
		businessMessage.Use(middlewareGroup.BusinessGetUserMiddleware)
		businessMessage.Handle(utils.WithProm("handleMessage", handlerGroup.HandleMessage), th.AnyBusinessMessage())
	}
}

func (bm *BotManager) processBotUpdates(ctx context.Context, instance *BotInstance) {
	for {
		select {
		case <-ctx.Done():
			log.Debug().Int64("botID", instance.ID).Msg("bot update processing stopped")
			return
		case _, ok := <-instance.Updates:
			if !ok {
				log.Debug().Int64("botID", instance.ID).Msg("updates channel closed")
				return
			}
		}
	}
}
