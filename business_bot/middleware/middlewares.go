package middleware

import (
	"context"
	"errors"
	"ssuspy-bot/locales"
	"ssuspy-bot/prom"
	"ssuspy-bot/redis"
	"ssuspy-bot/repository"
	"ssuspy-bot/types"
	"ssuspy-bot/utils"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type MiddlewareGroup struct {
	service *repository.MongoRepository
	rdb     *redis.Redis
}

func NewMiddlewareGroup(service *repository.MongoRepository, rdb *redis.Redis) *MiddlewareGroup {
	return &MiddlewareGroup{
		service: service,
		rdb:     rdb,
	}
}

func (h *MiddlewareGroup) GetInternalUserMiddleware(c *th.Context, update telego.Update) error {
	botID := c.Value("botID").(int64)
	var (
		iUser        *repository.IUser
		internalUser types.InternalUser
	)

	switch {
	case update.Message != nil:
		internalUser = types.InternalUser{
			ID:           update.Message.From.ID,
			FirstName:    update.Message.From.FirstName,
			LastName:     update.Message.From.LastName,
			LanguageCode: update.Message.From.LanguageCode,
			SendMessages: true,
		}
	case update.CallbackQuery != nil:
		internalUser = types.InternalUser{
			ID:           update.CallbackQuery.From.ID,
			FirstName:    update.CallbackQuery.From.FirstName,
			LastName:     update.CallbackQuery.From.LastName,
			LanguageCode: update.CallbackQuery.From.LanguageCode,
		}
	case update.BusinessConnection != nil:
		internalUser = types.InternalUser{
			ID:                   update.BusinessConnection.User.ID,
			FirstName:            update.BusinessConnection.User.FirstName,
			LastName:             update.BusinessConnection.User.LastName,
			LanguageCode:         update.BusinessConnection.User.LanguageCode,
			BusinessConnectionID: update.BusinessConnection.ID,
			SendMessages:         true,
		}
	case update.BusinessMessage != nil:
		iUser = utils.ProcessBusinessBot(h.service, update.BusinessMessage.BusinessConnectionID, 0, botID)
		if iUser == nil {
			return nil
		}

		internalUser = types.InternalUser{
			ID:                   iUser.User.ID,
			LanguageCode:         iUser.User.LanguageCode,
			BusinessConnectionID: update.BusinessMessage.BusinessConnectionID,
			SendMessages:         iUser.BotUser.SendMessages,
		}
	case update.DeletedBusinessMessages != nil:
		iUser = utils.ProcessBusinessBot(h.service, update.DeletedBusinessMessages.BusinessConnectionID, 0, botID)
		if iUser == nil {
			return nil
		}

		internalUser = types.InternalUser{
			ID:                   iUser.User.ID,
			LanguageCode:         iUser.User.LanguageCode,
			BusinessConnectionID: update.DeletedBusinessMessages.BusinessConnectionID,
			SendMessages:         iUser.BotUser.SendMessages,
		}
	case update.EditedBusinessMessage != nil:
		iUser = utils.ProcessBusinessBot(h.service, update.EditedBusinessMessage.BusinessConnectionID, 0, botID)
		if iUser == nil {
			return nil
		}

		internalUser = types.InternalUser{
			ID:                   iUser.User.ID,
			LanguageCode:         iUser.User.LanguageCode,
			BusinessConnectionID: update.EditedBusinessMessage.BusinessConnectionID,
			SendMessages:         iUser.BotUser.SendMessages,
		}
	case update.MyChatMember != nil:
		internalUser = types.InternalUser{
			ID:           update.MyChatMember.From.ID,
			FirstName:    update.MyChatMember.From.FirstName,
			LastName:     update.MyChatMember.From.LastName,
			LanguageCode: update.MyChatMember.From.LanguageCode,
		}
	case update.InlineQuery != nil:
		internalUser = types.InternalUser{
			ID:           update.InlineQuery.From.ID,
			FirstName:    update.InlineQuery.From.FirstName,
			LastName:     update.InlineQuery.From.LastName,
			LanguageCode: update.InlineQuery.From.LanguageCode,
		}
	case update.ChosenInlineResult != nil:
		internalUser = types.InternalUser{
			ID:           update.ChosenInlineResult.From.ID,
			FirstName:    update.ChosenInlineResult.From.FirstName,
			LastName:     update.ChosenInlineResult.From.LastName,
			LanguageCode: update.ChosenInlineResult.From.LanguageCode,
		}
	default:
		return errors.New("userID not found")
	}

	c = c.WithValue("iUser", iUser)
	c = c.WithValue("internalUser", &internalUser)

	logger := log.With().Int64("userID", internalUser.ID).Logger()
	c = c.WithValue("log", &logger)

	return c.Next(update)
}

func (h *MiddlewareGroup) SyncUserMiddleware(c *th.Context, update telego.Update) error {
	botID := c.Value("botID").(int64)
	log := c.Value("log").(*zerolog.Logger)
	internalUser := c.Value("internalUser").(*types.InternalUser)

	i18nLang := internalUser.LanguageCode
	if internalUser.LanguageCode == "" {
		i18nLang = "en"
	}

	iUser, new, err := h.service.FindOrCreateIUser(context.TODO(), internalUser.ID, botID, internalUser.LanguageCode)
	if err != nil {
		log.Warn().Err(err).Int64("userID", internalUser.ID).Msg("failed get data")
		return err
	}

	if !new {
		if iUser.User.LanguageCode != "" {
			i18nLang = iUser.User.LanguageCode
		}
	}

	loc := locales.NewLocalizer(i18nLang)
	c = c.WithValue("loc", loc)
	c = c.WithValue("languageCode", i18nLang)
	c = c.WithValue("iUser", iUser)

	return c.Next(update)
}

func (h *MiddlewareGroup) BotContextMiddleware(botID int64) th.Handler {
	return func(c *th.Context, update telego.Update) error {
		c = c.WithValue("botID", botID)
		return c.Next(update)
	}
}

func PromMiddleware(c *th.Context, update telego.Update) error {
	handlerName := c.Value("handlerName").(string)

	start := time.Now()
	prom.RequestsTotal.WithLabelValues(handlerName).Inc()

	defer func() {
		duration := time.Since(start).Seconds()
		prom.ProcessingTime.WithLabelValues(handlerName).Observe(duration)
	}()

	err := c.Next(update)
	if err != nil {
		prom.ErrorsTotal.WithLabelValues(handlerName).Inc()
	}

	return err
}
