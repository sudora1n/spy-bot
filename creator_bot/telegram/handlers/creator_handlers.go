package handlers

import (
	"context"
	"fmt"
	managerv1 "ssuspy-proto/gen/manager/v1"

	"ssuspy-creator-bot/config"
	"ssuspy-creator-bot/consts"
	"ssuspy-creator-bot/telegram/callbacks"
	"ssuspy-creator-bot/telegram/keyboard"
	"ssuspy-creator-bot/telegram/utils"
	"ssuspy-creator-bot/types"

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
	if !botUser.CanConnectToBusiness || !botUser.SupportsInlineQueries {
		var messages []string
		if !botUser.CanConnectToBusiness {
			messages = append(messages, "noBusiness")
		}
		if !botUser.SupportsInlineQueries {
			messages = append(messages, "noInline")
		}

		for _, message := range messages {
			_, err = c.Bot().SendMessage(
				c,
				tu.Message(
					tu.ID(internalUser.ID),
					loc.MustLocalize(&i18n.LocalizeConfig{
						MessageID: fmt.Sprintf("errors.%s", message),
					}),
				).WithReplyMarkup(tu.InlineKeyboard(
					keyboard.ButtonsToRows(keyboard.BuildInstructionsKeyboardRows(loc))...,
				)))
			if err != nil {
				return err
			}
		}
		return nil
	}

	err = h.service.InsertBot(c, botUser.ID, internalUser.ID, message.Text, botUser.Username)
	if err != nil {
		return onCreatingBotFail(c, loc, internalUser.ID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	resp, err := h.grpcClient.CreateBot(ctx, &managerv1.CreateBotRequest{Id: botUser.ID})
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
	)).WithParseMode(telego.ModeHTML))

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
		).WithCallbackData(data.String()))

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
			).WithCallbackData(data.String()))
		}

		rows = append(rows, row)
	}

	var noBots bool
	if len(rows) == 0 {
		noBots = true
	}

	rows = append(rows, tu.InlineKeyboardRow(tu.InlineKeyboardButton(
		loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "backToMainMenu",
		}),
	).WithCallbackData(consts.CALLBACK_PREFIX_BACK_TO_START)))

	var text string
	if noBots {
		text = loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "handleBotsList.noBots",
		})
	} else {
		text = loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "handleBotsList.message",
		})
	}

	_, err = c.Bot().EditMessageText(
		c,
		tu.EditMessageText(tu.ID(internalUser.ID),
			query.Message.GetMessageID(),
			text,
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

	botInfo, err := h.service.FindBotWithUserCounts(c, internalUser.ID, data.BotID)
	if err != nil {
		log.Warn().Err(err).Msg("failed get data")
		utils.OnDataError(c, query.ID, loc)
		return err
	}

	removeData := types.HandleBotRemove{
		BotID: data.BotID,
	}

	_, err = c.Bot().EditMessageText(c, tu.EditMessageText(
		tu.ID(internalUser.ID),
		query.Message.GetMessageID(),
		loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "handleBotItem.message",
			TemplateData: map[string]any{
				"Username":      botInfo.Username,
				"Users":         botInfo.TotalUsers,
				"BusinessUsers": botInfo.TotalBusinessUsers,
			},
		}),
	).WithReplyMarkup(tu.InlineKeyboard(
		tu.InlineKeyboardRow(tu.InlineKeyboardButton(
			loc.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "handleBotItem.buttons.remove",
			}),
		).WithCallbackData(removeData.String())),
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

	resp, err := h.grpcClient.RemoveBot(c, &managerv1.RemoveBotRequest{Id: data.BotID})
	if err != nil {
		return onCreatingBotFail(c, loc, internalUser.ID)
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
			TemplateData: map[string]string{
				"Username": resp.GetUsername(),
			},
		}),
	).WithReplyMarkup(tu.InlineKeyboard(
		tu.InlineKeyboardRow(tu.InlineKeyboardButton(
			loc.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "handleBotItem.buttons.backToBotsList",
			}),
		).WithCallbackData(consts.CALLBACK_PREFIX_BOT_LIST)),
	)).WithParseMode(telego.ModeHTML))
	return err
}
