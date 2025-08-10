package middleware

import (
	"errors"
	"ssuspy-common/repository/mongoRepository"
	"time"

	"ssuspy-creator-bot/prom"
	"ssuspy-creator-bot/repository"
	"ssuspy-creator-bot/telegram/locales"
	"ssuspy-creator-bot/types"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
)

type MiddlewareGroup struct {
	repository *repository.Repository
}

func NewMiddlewareGroup(repository *repository.Repository) *MiddlewareGroup {
	return &MiddlewareGroup{
		repository: repository,
	}
}

func (h *MiddlewareGroup) GetInternalUserMiddleware(c *th.Context, update telego.Update) error {
	var (
		user         *mongoRepository.User
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

	user, err := h.repository.Mongo.FindUser(c, internalUser.ID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			err = h.repository.Mongo.CreateUser(c, internalUser.ID, i18nLang, true)
			if err != nil {
				log.Warn().Err(err).Int64("userID", internalUser.ID).Msg("failed create user")
				return err
			}
		}

		user, err = h.repository.Mongo.FindUser(c, internalUser.ID)
		if err != nil {
			log.Warn().Err(err).Int64("userID", internalUser.ID).Msg("failed get user")
			return err
		}
	}

	if user.LanguageCode != "" {
		i18nLang = user.LanguageCode
	}

	loc := locales.NewLocalizer(i18nLang)
	c = c.WithValue("loc", loc)
	c = c.WithValue("languageCode", i18nLang)
	// c = c.WithValue("user", user) // unused

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
