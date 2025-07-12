package handlers

import (
	"context"
	"fmt"
	managerv1 "ssuspy-proto/gen/manager/v1"

	"ssuspy-common/telegram/format"
	"ssuspy-creator-bot/config"
	"ssuspy-creator-bot/consts"
	"ssuspy-creator-bot/repository"
	"ssuspy-creator-bot/telegram/keyboard"
	"ssuspy-creator-bot/telegram/locales"
	"ssuspy-creator-bot/types"

	"strings"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
)

type Handler struct {
	service    *repository.MongoRepository
	grpcClient managerv1.ManagerServiceClient
}

func NewHandlerGroup(service *repository.MongoRepository, grpcClient managerv1.ManagerServiceClient) *Handler {
	return &Handler{
		service:    service,
		grpcClient: grpcClient,
	}
}

func buildStartText(loc *i18n.Localizer, firstName string, lastName string) string {
	name := format.Name(firstName, lastName)

	return loc.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "start.message",
		TemplateData: map[string]any{
			"Name": name,
		},
	})
}

func buildStartReplyMarkup(loc *i18n.Localizer) *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(
				loc.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "botList",
				}),
			).WithCallbackData(consts.CALLBACK_PREFIX_BOT_LIST),

			tu.InlineKeyboardButton(
				loc.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "start.buttons.language",
				}),
			).WithCallbackData(consts.CALLBACK_PREFIX_LANG),
		),
		tu.InlineKeyboardRow(
			keyboard.BuildInstructionsKeyboardRows(loc)...,
		),
	)
}

func HandleStart(c *th.Context, update telego.Update) error {
	loc := c.Value("loc").(*i18n.Localizer)
	internalUser := c.Value("internalUser").(*types.InternalUser)

	var (
		queryID   string
		messageID int
	)

	if update.CallbackQuery != nil {
		queryID, messageID = update.CallbackQuery.ID, update.CallbackQuery.Message.GetMessageID()
	}

	text := buildStartText(loc, internalUser.FirstName, internalUser.LastName)
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

func (h *Handler) HandleLanguageChange(c *th.Context, update telego.Update) error {
	query := update.CallbackQuery
	internalUser := c.Value("internalUser").(*types.InternalUser)

	parts := strings.Split(query.Data, "|")

	err := h.service.UpdateUserLanguage(context.Background(), query.From.ID, parts[1])
	if err != nil {
		return err
	}

	loc := locales.NewLocalizer(parts[1])

	text := buildStartText(loc, internalUser.FirstName, internalUser.LastName)
	replyMarkup := buildStartReplyMarkup(loc)
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
					).WithURL(config.Config.CreatorGithubURL),
				),
			),
		).WithParseMode(telego.ModeHTML))
	return err
}

func (h *Handler) HandleBlocked(_ *th.Context, update telego.Update) error {
	chatMember := update.MyChatMember
	if chatMember.NewChatMember.MemberStatus() == telego.MemberStatusBanned &&
		chatMember.Chat.Type == "private" {
		return h.service.UpdateUserSendMessages(context.Background(), chatMember.From.ID, false)
	}
	return nil
}
