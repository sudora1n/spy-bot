package handlers

import (
	"context"
	"fmt"
	"ssuspy-creator-bot/callbacks"
	"ssuspy-creator-bot/config"
	"ssuspy-creator-bot/consts"
	pb "ssuspy-creator-bot/pb"
	"ssuspy-creator-bot/types"
	"ssuspy-creator-bot/utils"

	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
)

func onCreatingBotFail(c *th.Context, loc *i18n.Localizer, userID int64) error {
	_, err := c.Bot().SendMessage(c, tu.Message(tu.ID(userID), loc.MustLocalize(
		&i18n.LocalizeConfig{
			MessageID: "errors.failedCreateBot",
		}),
	))
	return err
}

func (h *Handler) HandleToken(c *th.Context, update telego.Update) error {
	message := update.Message
	loc := c.Value("loc").(*i18n.Localizer)
	internalUser := c.Value("internalUser").(*types.InternalUser)

	c.Bot().DeleteMessage(c, &telego.DeleteMessageParams{
		ChatID:    tu.ID(internalUser.ID),
		MessageID: message.MessageID,
	})

	botsLen, err := h.service.LenBots(c, internalUser.ID)
	if err != nil {
		log.Warn().Err(err).Int64("userID", internalUser.ID).Msg("failed get len of bots")
		return onCreatingBotFail(c, loc, internalUser.ID)
	}

	if botsLen > config.Config.MaxBotsByUser {
		_, err := c.Bot().SendMessage(c, tu.Message(tu.ID(internalUser.ID), loc.MustLocalize(
			&i18n.LocalizeConfig{
				MessageID: "errors.tooManyBots",
			}),
		))
		return err
	}

	botExists, err := h.service.FindBotByToken(c, internalUser.ID, message.Text)
	if err != nil && err != mongo.ErrNoDocuments {
		return onCreatingBotFail(c, loc, internalUser.ID)
	}
	if botExists != nil {
		_, err := c.Bot().SendMessage(c, tu.Message(tu.ID(internalUser.ID), loc.MustLocalize(
			&i18n.LocalizeConfig{
				MessageID: "errors.botExists",
			}),
		))
		return err
	}

	newBot, err := telego.NewBot(message.Text, telego.WithAPIServer(config.Config.TelegramBot.ApiURL))
	if err != nil {
		if err == telego.ErrInvalidToken {
			_, err := c.Bot().SendMessage(c, tu.Message(tu.ID(internalUser.ID), loc.MustLocalize(
				&i18n.LocalizeConfig{
					MessageID: "errors.noMatch",
				}),
			))
			return err
		}
		return onCreatingBotFail(c, loc, internalUser.ID)
	}

	botUser, err := newBot.GetMe(c)
	if err != nil {
		return onCreatingBotFail(c, loc, internalUser.ID)
	}
	if !botUser.CanConnectToBusiness {
		_, err = c.Bot().SendMessage(c, tu.Message(tu.ID(internalUser.ID), loc.MustLocalize(
			&i18n.LocalizeConfig{
				MessageID: "errors.noBusiness.message",
			}),
		))
		return err
	}

	err = h.service.InsertBot(c, botUser.ID, internalUser.ID, message.Text, botUser.Username)
	if err != nil {
		return onCreatingBotFail(c, loc, internalUser.ID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	resp, err := h.grpcClient.AddBot(ctx, &pb.AddBotRequest{Id: botUser.ID})
	if err != nil {
		return onCreatingBotFail(c, loc, internalUser.ID)
	}

	_, err = c.Bot().SendMessage(c, tu.Message(tu.ID(internalUser.ID), loc.MustLocalize(
		&i18n.LocalizeConfig{
			MessageID: "handleToken",
			TemplateData: map[string]string{
				"Username": resp.GetUsername(),
			},
		}),
	).WithReplyMarkup(tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(
				loc.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "backToMainMenu",
				}),
			).WithCallbackData(consts.CALLBACK_PREFIX_BACK_TO_START),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(
				loc.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "botList",
				}),
			).WithCallbackData(consts.CALLBACK_PREFIX_BOT_LIST),
		),
	)))

	return err
}

func (h *Handler) HandleBotsList(c *th.Context, update telego.Update) error {
	query := update.CallbackQuery
	loc := c.Value("loc").(*i18n.Localizer)
	log := c.Value("log").(*zerolog.Logger)
	internalUser := c.Value("internalUser").(*types.InternalUser)

	bots, err := h.service.FindBots(c, internalUser.ID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get bots")
		utils.OnDataError(c, query.ID, loc)
		return err
	}

	rows := make([][]telego.InlineKeyboardButton, 0, 5)
	for i := 0; i < len(bots); i += 2 {
		row := make([]telego.InlineKeyboardButton, 0, 2)
		data := types.HandleBotItem{
			BotID: bots[i].ID,
		}

		row = append(row, tu.InlineKeyboardButton(
			loc.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "handleBotsList.item",
				TemplateData: map[string]any{
					"Count":    i + 1,
					"Username": bots[i].Username,
				},
			}),
		).WithCallbackData(fmt.Sprint(data)))

		if i+1 < len(bots) {
			data.BotID = bots[i+1].ID
			row = append(row, tu.InlineKeyboardButton(
				loc.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "handleBotsList.item",
					TemplateData: map[string]any{
						"Count":    i + 2,
						"Username": bots[i+1].Username,
					},
				}),
			).WithCallbackData(fmt.Sprint(data)))
		}

		rows = append(rows, row)
	}

	rows = append(rows, tu.InlineKeyboardRow(tu.InlineKeyboardButton(
		loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "backToMainMenu",
		}),
	).WithCallbackData(consts.CALLBACK_PREFIX_BACK_TO_START)))

	_, err = c.Bot().EditMessageText(
		c,
		tu.EditMessageText(tu.ID(internalUser.ID),
			query.Message.GetMessageID(),
			loc.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "handleBotsList.message",
			}),
		).WithReplyMarkup(tu.InlineKeyboard(rows...)),
	)

	return err
}

func (h *Handler) HandleBotItem(c *th.Context, update telego.Update) error {
	query := update.CallbackQuery
	loc := c.Value("loc").(*i18n.Localizer)
	log := c.Value("log").(*zerolog.Logger)
	internalUser := c.Value("internalUser").(*types.InternalUser)

	data, err := callbacks.NewHandleBotItemFromString(query.Data)
	if err != nil {
		log.Warn().Err(err).Msg("failed get data")
		utils.OnDataError(c, query.ID, loc)
		return err
	}

	botUser, err := h.service.FindBotByID(c, internalUser.ID, data.BotID)
	if err != nil {
		log.Warn().Err(err).Msg("failed get data")
		utils.OnDataError(c, query.ID, loc)
		return err
	}

	_, err = c.Bot().EditMessageText(c, tu.EditMessageText(
		tu.ID(internalUser.ID),
		query.Message.GetMessageID(),
		loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "handleBotItem.message",
			TemplateData: map[string]string{
				"Username": botUser.Username,
			},
		}),
	).WithReplyMarkup(tu.InlineKeyboard(
		tu.InlineKeyboardRow(tu.InlineKeyboardButton(
			loc.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "handleBotItem.buttons.remove",
			}),
		).WithCallbackData(consts.CALLBACK_PREFIX_BOT_REMOVE)),
		tu.InlineKeyboardRow(tu.InlineKeyboardButton(
			loc.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "handleBotItem.buttons.backToBotsList",
			}),
		).WithCallbackData(consts.CALLBACK_PREFIX_BOT_LIST)),
	)))
	return err
}

func (h *Handler) HandleBotRemove(c *th.Context, update telego.Update) error {
	query := update.CallbackQuery
	loc := c.Value("loc").(*i18n.Localizer)
	log := c.Value("log").(*zerolog.Logger)
	internalUser := c.Value("internalUser").(*types.InternalUser)

	data, err := callbacks.NewHandleBotItemFromString(query.Data)
	if err != nil {
		log.Warn().Err(err).Msg("failed get data")
		utils.OnDataError(c, query.ID, loc)
		return err
	}

	err = h.service.RemoveBot(c, internalUser.ID, data.BotID)
	if err != nil {
		log.Warn().Err(err).Msg("failed remove bot")
		utils.OnDataError(c, query.ID, loc)
		return err
	}

	_, err = c.Bot().EditMessageText(c, tu.EditMessageText(
		tu.ID(internalUser.ID),
		query.Message.GetMessageID(),
		loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "handleBotRemove",
		}),
	).WithReplyMarkup(tu.InlineKeyboard(
		tu.InlineKeyboardRow(tu.InlineKeyboardButton(
			loc.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "handleBotItem.buttons.backToBotsList",
			}),
		).WithCallbackData(consts.CALLBACK_PREFIX_BOT_LIST)),
	)))
	return err
}
