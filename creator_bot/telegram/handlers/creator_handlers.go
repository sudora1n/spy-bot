package handlers

import (
	"errors"
	"fmt"
	"strings"

	"ssuspy-creator-bot/consts"
	createBot "ssuspy-creator-bot/service/create_bot"
	"ssuspy-creator-bot/telegram/callbacks"
	"ssuspy-creator-bot/telegram/keyboard"
	"ssuspy-creator-bot/telegram/utils"
	"ssuspy-creator-bot/types"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/rs/zerolog"
)

var (
	ERRORS_NOT_FOUND = errors.New("code of create bot not found")
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

	username, err := createBot.CreateBot(c, h.repository, internalUser.ID, message.Text)

	var botErr *createBot.CreateBotError
	if errors.As(err, &botErr) {
		switch botErr.Code {
		case createBot.STATUS_CREATEBOT_ERROR_INTERNAL:
			return onCreatingBotFail(c, loc, internalUser.ID)
		case createBot.STATUS_CREATEBOT_ERROR_TOO_MANY_BOTS:
			_, err := c.Bot().SendMessage(c, tu.Message(tu.ID(internalUser.ID), loc.MustLocalize(
				&i18n.LocalizeConfig{
					MessageID: "errors.tooManyBots",
				}),
			))

			return err
		case createBot.STATUS_CREATEBOT_ERROR_BOT_ALREADY_EXISTS:
			_, err := c.Bot().SendMessage(c, tu.Message(tu.ID(internalUser.ID), loc.MustLocalize(
				&i18n.LocalizeConfig{
					MessageID: "errors.botExists",
				}),
			))

			return err
		case createBot.STATUS_CREATEBOT_ERROR_BOT_INVALID:
			_, err := c.Bot().SendMessage(c, tu.Message(tu.ID(internalUser.ID), loc.MustLocalize(
				&i18n.LocalizeConfig{
					MessageID: "errors.noMatch",
				}),
			))

			return err
		case createBot.STATUS_CREATEBOT_ERROR_BOT_INVALID_SETTINGS:
			var errStrings []string

			for _, violation := range botErr.InvalidSettings {
				errStrings = append(
					errStrings,
					loc.MustLocalize(&i18n.LocalizeConfig{
						MessageID: fmt.Sprintf("errors.%s", violation),
					}),
				)
			}

			_, err := c.Bot().SendMessage(
				c,
				tu.Message(
					tu.ID(internalUser.ID),
					loc.MustLocalize(&i18n.LocalizeConfig{
						MessageID: "errors.botNotMatch",
						TemplateData: map[string]string{
							"Errors": strings.Join(errStrings, "\n"),
						},
					}),
				).WithReplyMarkup(tu.InlineKeyboard(
					keyboard.ButtonsToRows(keyboard.BuildInstructionsKeyboardRows(loc))...,
				)))

			return err
		}
	}

	_, err = c.Bot().SendMessage(c, tu.Message(tu.ID(internalUser.ID), loc.MustLocalize(
		&i18n.LocalizeConfig{
			MessageID: "handleToken",
			TemplateData: map[string]string{
				"Username": username,
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

	bots, err := h.repository.Mongo.FindBots(c, internalUser.ID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get bots")
		utils.OnDataError(c, query.ID, loc)
		return err
	}

	rows := make([][]telego.InlineKeyboardButton, 0, 5)
	for i := 0; i < len(bots); i += 2 {
		row := make([]telego.InlineKeyboardButton, 0, 2)
		data := types.HandleBotItem{
			BotID: bots[i].Id,
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
			data.BotID = bots[i+1].Id
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
	botStat, err := h.repository.Mongo.FindBotWithUserCounts(c, internalUser.ID, data.BotID)
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
				"Username":      botStat.Bot.Username,
				"Users":         botStat.TotalUsers,
				"BusinessUsers": botStat.TotalBusinessUsers,
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

	bot, err := h.repository.Mongo.BotByID(c, data.BotID)
	if err != nil {
		return onCreatingBotFail(c, loc, internalUser.ID)
	}

	if bot.UserID != internalUser.ID {
		return onCreatingBotFail(c, loc, internalUser.ID)
	}

	err = h.repository.Mongo.RemoveBot(c, internalUser.ID, data.BotID)
	if err != nil {
		return onCreatingBotFail(c, loc, internalUser.ID)
	}

	_, err = c.Bot().EditMessageText(c, tu.EditMessageText(
		tu.ID(internalUser.ID),
		query.Message.GetMessageID(),
		loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "handleBotRemove",
			TemplateData: map[string]string{
				"Username": bot.Username,
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
