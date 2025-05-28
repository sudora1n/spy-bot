package middleware

import (
	"errors"
	"ssuspy-bot/locales"
	"ssuspy-bot/repository"
	"ssuspy-bot/types"
	"ssuspy-bot/utils"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/rs/zerolog"
)

func (h *MiddlewareGroup) BusinessGetUserMiddleware(ctx *th.Context, update telego.Update) (err error) {
	log := ctx.Value("log").(*zerolog.Logger)
	iUser := ctx.Value("iUser").(*repository.IUser)
	internalUser := ctx.Value("internalUser").(*types.InternalUser)
	botID := ctx.Value("botID").(int64)

	var (
		messageIDs       []int
		chatID           int64
		dataID           int64
		itsCallbackQuery bool
	)

	{
		switch {
		case update.BusinessMessage != nil:
			chatID =
				update.BusinessMessage.Chat.ID
			break
		case update.DeletedBusinessMessages != nil:
			messageIDs, chatID =
				update.DeletedBusinessMessages.MessageIDs, update.DeletedBusinessMessages.Chat.ID
			break
		case update.EditedBusinessMessage != nil:
			chatID =
				update.EditedBusinessMessage.Chat.ID
			break
		case update.CallbackQuery != nil:
			itsCallbackQuery = true

			iUser = utils.ProcessBusinessBot(h.service, "", update.CallbackQuery.From.ID, botID)
			if iUser == nil {
				return nil
			}

			ctx = ctx.WithValue("iUser", iUser)
		default:
			return errors.New("unsupported update type.")
		}
	}

	ctx = ctx.WithValue("dataID", dataID)
	ctx = ctx.WithValue("itsCallbackQuery", itsCallbackQuery)
	ctx = ctx.WithValue("chatID", chatID)
	ctx = ctx.WithValue("messageIDs", messageIDs)

	var loc *i18n.Localizer
	if iUser == nil {
		loc = locales.NewLocalizer(internalUser.LanguageCode)
	} else {
		loc = locales.NewLocalizer(iUser.User.LanguageCode)
	}
	ctx = ctx.WithValue("loc", loc)

	logger := log.With().Int64("chatID", chatID).Logger()
	ctx = ctx.WithValue("log", &logger)

	return ctx.Next(update)
}
