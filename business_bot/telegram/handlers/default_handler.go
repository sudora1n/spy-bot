package handlers

import (
	"context"
	"fmt"
	"ssuspy-bot/config"
	"ssuspy-bot/consts"
	"ssuspy-bot/repository"
	"ssuspy-bot/telegram/format"
	"ssuspy-bot/telegram/keyboard"
	"ssuspy-bot/telegram/locales"
	"ssuspy-bot/types"

	"strings"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
)

func buildStartText(loc *i18n.Localizer, firstName string, lastName string, isConnected bool) string {
	name := format.Name(firstName, lastName)

	return loc.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "start.message",
		TemplateData: map[string]any{
			"Name":    name,
			"Enabled": isConnected,
		},
	})
}

func buildStartReplyMarkup(loc *i18n.Localizer, isConnected bool) *telego.InlineKeyboardMarkup {
	rows := make([][]telego.InlineKeyboardButton, 0, 3)
	rows = append(rows, tu.InlineKeyboardRow(
		tu.InlineKeyboardButton(
			loc.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "start.buttons.language",
			}),
		).WithCallbackData(consts.CALLBACK_PREFIX_LANG),
	))
	if isConnected {
		rows = append(rows, tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(
				loc.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "start.buttons.settings",
				}),
			).WithCallbackData(consts.CALLBACK_PREFIX_SETTINGS),
		))
	} else {
		helpRows := keyboard.BuildOnNewReplyMarkup(loc)
		rows = append(rows, helpRows...)
	}
	return tu.InlineKeyboard(rows...)
}

func HandleStart(c *th.Context, update telego.Update) error {
	loc := c.Value("loc").(*i18n.Localizer)
	iUser := c.Value("iUser").(*repository.IUser)
	internalUser := c.Value("internalUser").(*types.InternalUser)

	var (
		queryID   string
		messageID int
	)

	if update.CallbackQuery != nil {
		queryID, messageID = update.CallbackQuery.ID, update.CallbackQuery.Message.GetMessageID()
	}

	connection := iUser.BotUser.GetUserCurrentConnection()
	enabled := false
	if connection != nil {
		enabled = connection.Enabled
	}

	text := buildStartText(loc, internalUser.FirstName, internalUser.LastName, enabled)
	replyMarkup := buildStartReplyMarkup(loc, enabled)
	if queryID == "" {
		_, err := c.Bot().SendMessage(c, tu.Message(
			tu.ID(internalUser.ID),
			text,
		).WithReplyMarkup(replyMarkup).WithParseMode(telego.ModeHTML))

		return err
	}

	_, err := c.Bot().EditMessageText(c, tu.EditMessageText(
		tu.ID(internalUser.ID),
		messageID,
		text,
	).WithReplyMarkup(replyMarkup).WithParseMode(telego.ModeHTML))
	if err != nil {
		return err
	}
	return c.Bot().AnswerCallbackQuery(c, tu.CallbackQuery(queryID))
}

func HandleLanguage(c *th.Context, update telego.Update) error {
	query := update.CallbackQuery
	loc := c.Value("loc").(*i18n.Localizer)
	lang := c.Value("languageCode").(string)

	var rows [][]telego.InlineKeyboardButton
	userLang, err := language.Parse(lang)
	if err != nil {
		userLang = language.English
	}

	tags := display.Tags(userLang)
	for _, i18nTag := range locales.Bundle.LanguageTags() {
		langName := tags.Name(i18nTag)
		rows = append(
			rows,
			tu.InlineKeyboardRow(tu.InlineKeyboardButton(langName).
				WithCallbackData(fmt.Sprintf("%s|%s", consts.CALLBACK_PREFIX_LANG_CHANGE, i18nTag)),
			))
	}
	rows = append(rows, tu.InlineKeyboardRow(keyboard.BuildBackButton(loc, consts.CALLBACK_PREFIX_BACK_TO_START)))

	_, err = c.Bot().EditMessageText(c, tu.EditMessageText(
		tu.ID(query.From.ID),
		query.Message.GetMessageID(),
		loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "language",
		}),
	).WithReplyMarkup(tu.InlineKeyboard(rows...)))
	return err
}

func (h *Handler) HandleLanguageChange(c *th.Context, update telego.Update) error {
	query := update.CallbackQuery
	iUser := c.Value("iUser").(*repository.IUser)
	internalUser := c.Value("internalUser").(*types.InternalUser)

	parts := strings.Split(query.Data, "|")

	err := h.service.UpdateUserLanguage(context.Background(), query.From.ID, parts[1])
	if err != nil {
		return err
	}

	loc := locales.NewLocalizer(parts[1])

	connection := iUser.BotUser.GetUserCurrentConnection()
	enabled := false
	if connection != nil {
		enabled = connection.Enabled
	}

	text := buildStartText(loc, internalUser.FirstName, internalUser.LastName, enabled)
	replyMarkup := buildStartReplyMarkup(loc, enabled)
	_, err = c.Bot().EditMessageText(c, tu.EditMessageText(
		tu.ID(query.From.ID),
		query.Message.GetMessageID(),
		text,
	).WithReplyMarkup(replyMarkup).WithParseMode(telego.ModeHTML))
	return err
}

func HandleGithub(c *th.Context, update telego.Update) error {
	message := update.Message
	loc := c.Value("loc").(*i18n.Localizer)

	_, err := c.Bot().SendMessage(c, tu.Message(
		tu.ID(message.From.ID),
		loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "github.message",
		}),
	).
		WithReplyMarkup(
			tu.InlineKeyboard(
				tu.InlineKeyboardRow(
					tu.InlineKeyboardButton(
						loc.MustLocalize(&i18n.LocalizeConfig{
							MessageID: "github.buttons.open",
						}),
					).WithURL(config.Config.BusinessGithubURL),
				),
			),
		).WithParseMode(telego.ModeHTML))
	return err
}

func (h *Handler) HandleBlocked(c *th.Context, update telego.Update) error {
	chatMember := update.MyChatMember
	botID := c.Value("botID").(int64)
	internalUser := c.Value("internalUser").(*types.InternalUser)

	if chatMember.NewChatMember.MemberStatus() == telego.MemberStatusBanned &&
		chatMember.Chat.Type == "private" {
		return h.service.UpdateBotUserSendMessages(context.Background(), internalUser.ID, botID, false)
	}
	return nil
}
