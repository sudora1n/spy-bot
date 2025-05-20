package handlers

import (
	"context"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"ssuspy-bot/consts"
	"ssuspy-bot/format"
	"ssuspy-bot/redis"
	"ssuspy-bot/repository"
	"ssuspy-bot/types"
	"ssuspy-bot/utils"
)

func (h *Handler) HandleMessage(c *th.Context, message telego.Message) error {
	err := h.service.SaveMessage(context.Background(), message)
	if err != nil {
		log.Warn().
			Err(err).
			Int64("chatID", message.Chat.ID).
			Int("messageID", message.MessageID).
			Msg("error saving message")
		return nil
	}

	replyToMessage := message.ReplyToMessage
	if replyToMessage == nil {
		return nil
	}

	if !replyToMessage.HasProtectedContent {
		return nil
	}

	loc := c.Value("loc").(*i18n.Localizer)
	user := c.Value("user").(*repository.User)

	file := utils.GetFile(replyToMessage)
	if file == nil {
		return nil
	}

	protectedMessage, err := c.Bot().SendMessage(c, tu.Message(
		tu.ID(user.ID),
		loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "business.restrictedMedia",
		}),
	).WithParseMode(telego.ModeHTML))
	if err != nil {
		return err
	}

	err = h.rdb.EnqueueJob(c, consts.REDIS_QUEUE_FILES, redis.Job{
		Loc:       loc,
		File:      file,
		UserID:    user.ID,
		ChatID:    message.Chat.ID,
		MessageID: protectedMessage.MessageID,
		Caption:   replyToMessage.Caption,
	})
	if err != nil {
		return err
	}

	return err
}

func (h *Handler) HandleDeleted(c *th.Context, update telego.Update) error {
	loc := c.Value("loc").(*i18n.Localizer)
	user := c.Value("user").(*repository.User)
	log := c.Value("log").(*zerolog.Logger)

	itsCallbackQuery := c.Value("itsCallbackQuery").(bool)
	chatID := c.Value("chatID").(int64)
	messageIDs := c.Value("messageIDs").([]int)

	oldMsgs, pagination, err := h.service.GetMessages(
		context.Background(),
		&repository.GetMessagesOptions{
			ChatID:        chatID,
			MessageIDs:    messageIDs,
			ConnectionIDs: user.GetUserCurrentConnectionIDs(),
		},
	)
	if err != nil {
		return err
	}

	if len(oldMsgs) > consts.MAX_BUTTONS {
		pagination.Forward = true
	}

	if len(oldMsgs) == 0 {
		log.Warn().Ints("messageIDs", messageIDs).Msg("no messages found in the database")
		return nil
	}

	summaryText := format.SummarizeDeletedMessages(oldMsgs, loc)
	tempText := strings.ReplaceAll(summaryText, "\n", " ")
	if len(tempText) > consts.MAX_LEN {
		description := loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "business.deleted.overflowDescription",
		})
		summaryText = summaryText[:consts.MAX_LEN-len(strings.ReplaceAll(description, "\n", " "))] + description
	}

	var dataID int64
	if itsCallbackQuery {
		dataID = c.Value("dataID").(int64)
	} else {
		dataID, err = h.service.SetDataDeleted(context.TODO(), user.ID, messageIDs)
		if err != nil {
			return err
		}
	}

	rows := utils.DeletedRows(chatID, user, loc, oldMsgs, pagination, 0, dataID)

	if itsCallbackQuery {
		_, err = c.Bot().EditMessageText(c, tu.EditMessageText(
			tu.ID(user.ID),
			update.CallbackQuery.Message.GetMessageID(),
			summaryText,
		).
			WithParseMode(telego.ModeHTML).
			WithReplyMarkup(tu.InlineKeyboard(rows...)),
		)
		if err != nil {
			return err
		}

		return c.Bot().AnswerCallbackQuery(c, tu.CallbackQuery(update.CallbackQuery.ID))
	}
	_, err = c.Bot().SendMessage(c, tu.Message(
		tu.ID(user.ID),
		summaryText,
	).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(tu.InlineKeyboard(rows...)),
	)

	return err
}

func (h *Handler) HandleEdited(c *th.Context, message telego.Message) error {
	loc := c.Value("loc").(*i18n.Localizer)
	user := c.Value("user").(*repository.User)
	log := c.Value("log").(*zerolog.Logger)

	oldMsg, err := h.service.GetMessage(
		context.Background(),
		&repository.GetMessageOptions{
			ChatID:        message.Chat.ID,
			MessageID:     message.MessageID,
			ConnectionIDs: user.GetUserCurrentConnectionIDs(),
		},
	)
	if err != nil {
		log.Error().Err(err).
			Int("message_id", message.MessageID).
			Msg("failed GetMessage")
		errSave := h.service.SaveMessage(context.Background(), message)
		if errSave != nil {
			log.Error().Err(errSave).Msg("error saving edited business message after failing to retrieve old message")
		}
		return err
	}

	changes, mediaDiff := format.EditedDiff(oldMsg, &message, loc)

	if len(changes) == 0 {
		err = h.service.SaveMessage(context.Background(), message)
		if err != nil {
			log.Error().Err(err).
				Int("message_id", message.MessageID).
				Msg("error saving edited business message (no changes detected)")
		}
		return err
	}

	diffText := strings.Join(changes, "\n\n")
	editedAt := time.Unix(int64(message.EditDate), 0).Format(consts.DATETIME_FOR_MESSAGE)
	formattedText := loc.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "business.edited.message",
		TemplateData: map[string]any{
			"ChatID": message.Chat.ID,
			"Date":   editedAt,
			"Diff":   diffText,
		},
	})

	var (
		date       int64
		dateIsEdit bool
	)

	if oldMsg.EditDate == 0 {
		date = oldMsg.Date
	} else {
		date, dateIsEdit = oldMsg.EditDate, true
	}

	dataID, err := h.service.SetDataEdited(context.TODO(), &repository.SetDataEditedOptions{
		MessageID:     message.MessageID,
		UserID:        user.ID,
		OldDate:       date,
		OldDateIsEdit: dateIsEdit,
		NewDate:       message.EditDate,
	})
	if err != nil {
		return err
	}

	callbackData := types.HandleEditedData{
		DataID: dataID,
		ChatID: message.Chat.ID,
	}
	replyMarkup := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(
				loc.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "business.edited.buttons.log",
				}),
			).WithCallbackData(callbackData.ToString(types.HandleEditedDataTypeLog)),
		),
	)

	if mediaDiff.Removed != nil {
		replyMarkup.InlineKeyboard = append(replyMarkup.InlineKeyboard,
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton(
					loc.MustLocalize(&i18n.LocalizeConfig{
						MessageID: "business.edited.buttons.getFile",
					}),
				).WithCallbackData(callbackData.ToString(types.HandleEditedDataTypeFiles)),
			),
		)
	}

	if len(formattedText) <= consts.MAX_LEN {
		_, err = c.Bot().SendMessage(
			c,
			tu.Message(
				tu.ID(user.ID),
				formattedText,
			).WithParseMode(telego.ModeHTML).WithReplyMarkup(replyMarkup),
		)
	} else {
		_, err = c.Bot().SendMessage(c, tu.Message(
			tu.ID(user.ID),
			loc.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "business.edited.messageOverflow",
				TemplateData: map[string]any{
					"ChatID": message.Chat.ID,
					"Date":   editedAt,
				},
			}),
		).
			WithParseMode(telego.ModeHTML).WithReplyMarkup(replyMarkup))
	}

	errSave := h.service.SaveMessage(context.Background(), message)
	if errSave != nil {
		log.Error().Err(errSave).
			Int("message_id", message.MessageID).
			Msg("error saving edited message to database")
		return errSave
	}

	if err != nil {
		log.Error().Err(err).
			Int("message_id", message.MessageID).
			Msg("error sending edit notification")
	}

	return err
}

func (h *Handler) HandleConnection(c *th.Context, connection telego.BusinessConnection) error {
	loc := c.Value("loc").(*i18n.Localizer)

	err := h.service.UpdateUserConnection(context.Background(), &connection)
	if err != nil {
		return err
	}

	var text string
	if connection.IsEnabled {
		name := format.Name(connection.User.FirstName, connection.User.LastName)

		text = loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "business.connection.on",
			TemplateData: map[string]string{
				"Name": name,
			},
		})
	} else {
		text = loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "business.connection.off",
		})
	}

	_, err = c.Bot().SendMessage(c, tu.Message(
		tu.ID(connection.User.ID),
		text,
	).WithParseMode(telego.ModeHTML))
	return err
}
