package middleware

import (
	"context"
	"errors"
	"ssuspy-creator-bot/locales"
	"ssuspy-creator-bot/prom"
	"ssuspy-creator-bot/redis"
	"ssuspy-creator-bot/repository"
	"ssuspy-creator-bot/types"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
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
	var (
		user         *repository.User
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
	case update.MyChatMember != nil:
		internalUser = types.InternalUser{
			ID:           update.MyChatMember.From.ID,
			FirstName:    update.MyChatMember.From.FirstName,
			LastName:     update.MyChatMember.From.LastName,
			LanguageCode: update.MyChatMember.From.LanguageCode,
			SendMessages: false,
		}
	default:
		return errors.New("userID not found")
	}

	c = c.WithValue("user", user)
	c = c.WithValue("internalUser", &internalUser)

	logger := log.With().Int64("userID", internalUser.ID).Logger()
	c = c.WithValue("log", &logger)

	return c.Next(update)
}

func (h *MiddlewareGroup) SyncUserMiddleware(c *th.Context, update telego.Update) error {
	internalUser := c.Value("internalUser").(*types.InternalUser)

	i18nLang := "en"
	if internalUser.LanguageCode != "" {
		i18nLang = internalUser.LanguageCode
	}
	new, err := h.service.UpdateUser(context.TODO(), internalUser.ID, internalUser.LanguageCode, internalUser.SendMessages)
	if err != nil {
		return err
	}

	user, err := h.service.FindUser(context.TODO(), internalUser.ID)
	if err != nil {
		return err
	}

	if !new {
		if user.LanguageCode != "" {
			i18nLang = user.LanguageCode
		}
	}

	loc := locales.NewLocalizer(i18nLang)
	c = c.WithValue("loc", loc)
	c = c.WithValue("languageCode", i18nLang)
	c = c.WithValue("user", user)

	return c.Next(update)
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

func AutoRespond(c *th.Context, update telego.Update) error {
	if update.CallbackQuery != nil {
		defer func() {
			c.Bot().AnswerCallbackQuery(c, tu.CallbackQuery(update.CallbackQuery.ID))
		}()
	}

	return c.Next(update)
}
