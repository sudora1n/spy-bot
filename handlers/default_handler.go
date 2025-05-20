package handlers

import (
	"context"
	"fmt"
	bundlei18n "ssuspy-bot/bundle_i18n"
	"ssuspy-bot/config"
	"ssuspy-bot/consts"
	"ssuspy-bot/format"
	"ssuspy-bot/repository"
	"ssuspy-bot/types"

	"strings"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
)

func buildStartText(loc *i18n.Localizer, user *repository.User, firstName string, lastName string) string {
	name := format.Name(firstName, lastName)

	connection := user.GetUserCurrentConnection()
	enabled := false
	if connection != nil {
		enabled = connection.Enabled
	}

	return loc.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "start.message",
		TemplateData: map[string]any{
			"Name":    name,
			"Enabled": enabled,
		},
	})
}

func buildStartReplyMarkup(loc *i18n.Localizer) *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(
				loc.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "start.buttons.language",
				}),
			).WithCallbackData(consts.CALLBACK_PREFIX_LANG),
		),
	)
}

func HandleStart(c *th.Context, update telego.Update) error {
	loc := c.Value("loc").(*i18n.Localizer)
	user := c.Value("user").(*repository.User)
	internalUser := c.Value("internalUser").(*types.InternalUser)

	var (
		queryID   string
		messageID int
	)

	if update.CallbackQuery != nil {
		queryID, messageID = update.CallbackQuery.ID, update.CallbackQuery.Message.GetMessageID()
	}

	text := buildStartText(loc, user, internalUser.FirstName, internalUser.LastName)
	replyMarkup := buildStartReplyMarkup(loc)
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

func HandleLanguage(c *th.Context, query telego.CallbackQuery) error {
	loc := c.Value("loc").(*i18n.Localizer)
	lang := c.Value("languageCode").(string)

	var rows [][]telego.InlineKeyboardButton
	userLang, err := language.Parse(lang)
	if err != nil {
		userLang = language.English
	}

	tags := display.Tags(userLang)
	for _, i18nTag := range bundlei18n.Bundle.LanguageTags() {
		langName := tags.Name(i18nTag)
		rows = append(
			rows,
			tu.InlineKeyboardRow(tu.InlineKeyboardButton(langName).
				WithCallbackData(fmt.Sprintf("%s|%s", consts.CALLBACK_PREFIX_LANG_CHANGE, i18nTag)),
			))
	}
	rows = append(rows, tu.InlineKeyboardRow(tu.InlineKeyboardButton(
		loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "back",
		}),
	).WithCallbackData(consts.CALLBACK_PREFIX_BACK_TO_START)))

	_, err = c.Bot().EditMessageText(c, tu.EditMessageText(
		tu.ID(query.From.ID),
		query.Message.GetMessageID(),
		loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "language",
		}),
	).WithReplyMarkup(tu.InlineKeyboard(rows...)))
	return err
}

func (h *Handler) HandleLanguageChange(c *th.Context, query telego.CallbackQuery) error {
	user := c.Value("user").(*repository.User)
	internalUser := c.Value("internalUser").(*types.InternalUser)

	parts := strings.Split(query.Data, "|")

	err := h.service.UpdateUserLanguage(context.Background(), query.From.ID, parts[1])
	if err != nil {
		return err
	}

	loc := bundlei18n.NewLocalizer(parts[1])

	text := buildStartText(loc, user, internalUser.FirstName, internalUser.LastName)
	replyMarkup := buildStartReplyMarkup(loc)
	_, err = c.Bot().EditMessageText(c, tu.EditMessageText(
		tu.ID(query.From.ID),
		query.Message.GetMessageID(),
		text,
	).WithReplyMarkup(replyMarkup).WithParseMode(telego.ModeHTML))
	return err
}

func HandleGithub(c *th.Context, message telego.Message) error {
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
					).WithURL(config.Config.GithubURL),
				),
			),
		).WithParseMode(telego.ModeHTML))
	return err
}

func (h *Handler) HandleBlocked(_ *th.Context, chatMember telego.ChatMemberUpdated) error {
	if chatMember.NewChatMember.MemberStatus() == telego.MemberStatusBanned &&
		chatMember.Chat.Type == "private" {
		return h.service.UpdateUserSendMessages(context.Background(), chatMember.From.ID, false)
	}
	return nil
}
