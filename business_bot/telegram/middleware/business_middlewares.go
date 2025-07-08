package middleware

import (
	"errors"
	"ssuspy-bot/repository"
	"ssuspy-bot/telegram/locales"
	"ssuspy-bot/telegram/utils"
	"ssuspy-bot/types"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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
			messageIDs = update.DeletedBusinessMessages.MessageIDs

			filtered := make([]int, 0, len(messageIDs))
			for _, messageID := range messageIDs {
				ignore, err := h.rdb.IsMessageIgnore(
					ctx,
					messageID,
					update.DeletedBusinessMessages.Chat.ID,
				)
				if err != nil {
					log.Warn().Err(err).Msg("failed get ignore message")
				}
				if !ignore {
					filtered = append(filtered, messageID)
				}
			}

			if len(filtered) == 0 {
				return nil
			}

			messageIDs = filtered
			chatID = update.DeletedBusinessMessages.Chat.ID
			break
		case update.EditedBusinessMessage != nil:
			chatID = update.EditedBusinessMessage.Chat.ID

			ignore, err := h.rdb.IsMessageIgnore(
				ctx,
				update.EditedBusinessMessage.MessageID,
				chatID,
			)
			if err != nil {
				log.Warn().Err(err).Msg("failed get ignore message")
			}
			if ignore {
				return nil
			}

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

// func (h *MiddlewareGroup) BusinessIsIgnore(ctx *th.Context, update telego.Update) (err error) {
// 	log := ctx.Value("log").(*zerolog.Logger)

// 	if update.BusinessMessage != nil {
// 		ignore, err := h.rdb.IsMessageIgnore(
// 			ctx,
// 			update.BusinessMessage.MessageID,
// 			update.BusinessMessage.Chat.ID,
// 		)
// 		if err != nil {
// 			log.Warn().Err(err).Msg("failed get ignore message")
// 		}
// 		if ignore {
// 			return nil
// 		}
// 	}

// 	return ctx.Next(update)
// }

func (h *MiddlewareGroup) BusinessIsFromUser(ctx *th.Context, update telego.Update) (err error) {
	iUser := ctx.Value("iUser").(*repository.IUser)
	if update.BusinessMessage != nil && update.BusinessMessage.From.ID == iUser.User.ID {
		return ctx.Next(update)
	}
	return nil
}

func (h *MiddlewareGroup) BusinessUserSetRights(ctx *th.Context, update telego.Update) (err error) {
	iUser := ctx.Value("iUser").(*repository.IUser)

	connection := iUser.BotUser.GetUserCurrentConnection()
	rights, err := utils.GetBusinessRights(ctx, connection)
	if err != nil {
		log.Warn().Err(err).Msg("failed get business connection")
		return err
	}

	ctx = ctx.WithValue("rights", rights)
	ctx = ctx.WithValue("userConnection", connection)
	return ctx.Next(update)
}

func (h *MiddlewareGroup) BusinessIgnoreMessage(ctx *th.Context, update telego.Update) (err error) {
	log := ctx.Value("log").(*zerolog.Logger)

	if update.BusinessMessage != nil {
		err = h.rdb.IgnoreMessage(
			ctx,
			update.BusinessMessage.MessageID,
			update.BusinessMessage.Chat.ID,
		)
		if err != nil {
			log.Warn().Err(err).Msg("failed save message as ignore")
		}
	}

	return ctx.Next(update)
}
