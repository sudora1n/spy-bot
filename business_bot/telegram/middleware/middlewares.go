package middleware

import (
	"errors"
	"ssuspy-bot/repository"
	"ssuspy-bot/telegram/locales"
	"ssuspy-common/repository/mongoRepository"
	"ssuspy-common/types"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/rs/zerolog"
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

// because creator bot
func SkipNonPrivateChatsMiddleware(c *th.Context, update telego.Update) error {
	chatType := telego.ChatTypePrivate
	switch {
	case update.Message != nil:
		chatType = update.Message.Chat.Type
	case update.MyChatMember != nil:
		chatType = update.MyChatMember.Chat.Type
	}

	if chatType != telego.ChatTypePrivate {
		return nil
	}

	return c.Next(update)
}

func (h *MiddlewareGroup) GetInternalUserMiddleware(c *th.Context, update telego.Update) error {
	botID := c.Value("botID").(int64)
	var (
		internalUser types.InternalUser
		user         *mongoRepository.User
		botUser      *mongoRepository.BotUser
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
		userWithBotUser, err := h.repository.Mongo.FindIUserByConnectionID(c, update.BusinessMessage.BusinessConnectionID, botID, false)
		if userWithBotUser.BotUser == nil || err != nil {
			return nil
		}
		user, botUser = &userWithBotUser.User, userWithBotUser.BotUser

		internalUser = types.InternalUser{
			ID:                   user.Id,
			LanguageCode:         user.LanguageCode,
			BusinessConnectionID: update.BusinessMessage.BusinessConnectionID,
			SendMessages:         botUser.SendMessages,
		}
	case update.DeletedBusinessMessages != nil:
		userWithBotUser, err := h.repository.Mongo.FindIUserByConnectionID(
			c,
			update.DeletedBusinessMessages.BusinessConnectionID,
			botID,
			true,
		)
		if userWithBotUser.BotUser == nil || err != nil {
			return nil
		}
		user, botUser = &userWithBotUser.User, userWithBotUser.BotUser

		internalUser = types.InternalUser{
			ID:                   user.Id,
			LanguageCode:         user.LanguageCode,
			BusinessConnectionID: update.DeletedBusinessMessages.BusinessConnectionID,
			SendMessages:         botUser.SendMessages,
		}
	case update.EditedBusinessMessage != nil:
		userWithBotUser, err := h.repository.Mongo.FindIUserByConnectionID(
			c,
			update.EditedBusinessMessage.BusinessConnectionID,
			botID,
			true,
		)
		if userWithBotUser.BotUser == nil || err != nil {
			return nil
		}
		user, botUser = &userWithBotUser.User, userWithBotUser.BotUser

		internalUser = types.InternalUser{
			ID:                   user.Id,
			LanguageCode:         user.LanguageCode,
			BusinessConnectionID: update.EditedBusinessMessage.BusinessConnectionID,
			SendMessages:         botUser.SendMessages,
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

	c = c.WithValue("user", user)
	c = c.WithValue("botUser", botUser)
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

	var (
		user    *mongoRepository.User
		botUser *mongoRepository.BotUser
	)

	res, err := h.repository.Mongo.FindIUserByID(c, internalUser.ID, botID)
	user, botUser = &res.User, res.BotUser
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			err := h.repository.Mongo.CreateUser(c, internalUser.ID, i18nLang, false)
			if err != nil && !mongo.IsDuplicateKeyError(err) {
				log.Warn().Err(err).Int64("userID", internalUser.ID).Msg("failed create user")
				return err
			}
		}

		userRes, err := h.repository.Mongo.FindUser(c, internalUser.ID)
		if err != nil {
			log.Warn().Err(err).Int64("userID", internalUser.ID).Msg("failed get user")
			return err
		}

		err = h.repository.Mongo.CreateBotUser(c, internalUser.ID, botID)
		if err != nil {
			log.Warn().Err(err).Int64("userID", internalUser.ID).Msg("failed create bot user")
			return err
		}

		botUserRes, err := h.repository.Mongo.FindBotUser(c, internalUser.ID, botID)
		if err != nil {
			log.Warn().Err(err).Int64("userID", internalUser.ID).Msg("failed get bot user")
			return err
		}

		user, botUser = userRes, botUserRes
	}

	loc := locales.NewLocalizer(i18nLang)
	c = c.WithValue("loc", loc)
	c = c.WithValue("languageCode", i18nLang)
	c = c.WithValue("user", user)
	c = c.WithValue("botUser", botUser)

	return c.Next(update)
}

func (h *MiddlewareGroup) BotContextMiddleware(botID int64) th.Handler {
	return func(c *th.Context, update telego.Update) error {
		c = c.WithValue("botID", botID)
		return c.Next(update)
	}
}
