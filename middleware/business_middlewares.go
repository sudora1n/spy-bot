package middleware

import (
	"context"
	"errors"
	"fmt"
	"ssuspy-bot/callbacks"
	"ssuspy-bot/locales"
	"ssuspy-bot/repository"
	"ssuspy-bot/types"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/rs/zerolog"
)

func (h *MiddlewareGroup) BusinessGetUserMiddleware(ctx *th.Context, update telego.Update) (err error) {
	log := ctx.Value("log").(*zerolog.Logger)
	user := ctx.Value("user").(*repository.User)
	internalUser := ctx.Value("internalUser").(*types.InternalUser)

	var (
		messageIDs       []int
		chatID           int64
		dataID           int64
		itsCallbackQuery bool
		itsNewUser       bool
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
			user = h.processBusiness("", update.CallbackQuery.From.ID)
			if user == nil {
				return nil
			}

			itsNewUser = true

			data, err := callbacks.NewHandleBusinessDataFromString(update.CallbackQuery.Data)
			if err != nil {
				log.Warn().Err(err).Str("data", update.CallbackQuery.Data).Msg("invalid callback data")
				return fmt.Errorf("invalid callback data")
			}

			result, err := h.service.GetDataDeleted(context.Background(), update.CallbackQuery.From.ID, data.DataID)
			if err != nil {
				log.Error().Err(err).Int64("dataID", data.DataID).Msg("error GetDataFullDeletedLogByUUID")

				return err
			}
			messageIDs, chatID, itsCallbackQuery, dataID = result.MessageIDs, data.ChatID, true, data.DataID
		default:
			return errors.New("unsupported update type.")
		}
	}

	if itsNewUser {
		ctx = ctx.WithValue("user", user)
	}

	ctx = ctx.WithValue("dataID", dataID)
	ctx = ctx.WithValue("itsCallbackQuery", itsCallbackQuery)
	ctx = ctx.WithValue("chatID", chatID)
	ctx = ctx.WithValue("messageIDs", messageIDs)

	var loc *i18n.Localizer
	if user == nil {
		loc = locales.NewLocalizer(internalUser.LanguageCode)
	} else {
		loc = locales.NewLocalizer(user.LanguageCode)
	}
	ctx = ctx.WithValue("loc", loc)

	logger := log.With().Int64("chatID", chatID).Logger()
	ctx = ctx.WithValue("log", &logger)

	return ctx.Next(update)
}
