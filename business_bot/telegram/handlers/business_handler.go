package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"ssuspy-bot/consts"
	"ssuspy-bot/repository/redis"
	"ssuspy-bot/telegram/callbacks"
	lFormat "ssuspy-bot/telegram/format"
	"ssuspy-common/repository/mongoRepository"
	"ssuspy-common/telegram/format"
	"ssuspy-common/telegram/utils"
	commonTypes "ssuspy-common/types"
)

func (h *Handler) HandleMessage(c *th.Context, update telego.Update) error {
	message := update.BusinessMessage

	internalUser := c.Value("internalUser").(commonTypes.InternalUser)

	message.Text = format.TruncateText(message.Text, consts.MAX_USER_MESSAGE_TEXT_LEN, false)
	message.Caption = format.TruncateText(message.Caption, consts.MAX_USER_MESSAGE_TEXT_LEN, false)

	err := h.repo.Mongo.SaveMessage(context.Background(), message, internalUser.ID)
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

	botID := c.Value("botID").(int64)
	loc := c.Value("loc").(*i18n.Localizer)
	user := c.Value("user").(*mongoRepository.User)

	file := utils.GetFile(replyToMessage)
	if file == nil {
		return nil
	}

	fileExists, err := h.repo.Mongo.CreateFileIfNotExists(c, file.FileID, user.Id, message.Chat.ID)
	if err != nil || !fileExists {
		return err
	}

	protectedMessage, err := c.Bot().SendMessage(c, tu.Message(
		tu.ID(user.Id),
		loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "business.restrictedMedia",
		}),
	).WithParseMode(telego.ModeHTML))
	if err != nil {
		return err
	}

	err = h.repo.LRedis.EnqueueJob(c, consts.REDIS_QUEUE_FILES, redis.Job{
		File:             file,
		UserID:           user.Id,
		ChatID:           message.Chat.ID,
		MessageID:        protectedMessage.MessageID,
		Caption:          replyToMessage.Caption,
		UserLanguageCode: user.LanguageCode,
		BotID:            botID,
	})
	if err != nil {
		return err
	}

	return err
}

func (h *Handler) HandleDeleted(c *th.Context, update telego.Update) error {
	loc := c.Value("loc").(*i18n.Localizer)
	user := c.Value("user").(*mongoRepository.User)
	log := c.Value("log").(*zerolog.Logger)

	itsCallbackQuery := c.Value("itsCallbackQuery").(bool)
	chatID := c.Value("chatID").(int64)
	messageIDs := c.Value("messageIDs").([]int)

	var (
		limit              int
		offset             int
		typeOfPagination   string
		dataID             primitive.ObjectID
		correctMessagesLen uint8
		correctFilesLen    uint8
	)
	if itsCallbackQuery {
		data, err := callbacks.NewHandleDeletedPaginationDataFromString(update.CallbackQuery.Data)
		if err != nil {
			log.Warn().Err(err).Str("data", update.CallbackQuery.Data).Msg("invalid callback data")
			return fmt.Errorf("invalid callback data")
		}

		result, err := h.repo.Mongo.GetDataDeleted(context.Background(), update.CallbackQuery.From.ID, data.DataID)
		if err != nil {
			log.Error().Err(err).Str("dataID", data.DataID.Hex()).Msg("error GetDataFullDeletedLogByUUID")

			return err
		}

		messageIDs, chatID, typeOfPagination, offset, dataID, limit, correctMessagesLen, correctFilesLen =
			result.MessageIDs, data.ChatID, data.TypeOfPagination, data.Offset, data.DataID, consts.MAX_BUTTONS, result.MessagesCount, result.FilesCount
	}

	switch typeOfPagination {
	case "f":
		offset = offset + consts.MAX_BUTTONS
	case "b":
		offset = max(offset-consts.MAX_BUTTONS, 0)
	}

	msgRes, err := h.repo.Mongo.GetMessages(
		context.Background(),
		&mongoRepository.GetMessagesOptions{
			UserID:     user.Id,
			PeerID:     chatID,
			MessageIDs: messageIDs,
			Limit:      limit,
			Offset:     offset,
		},
	)
	if err != nil {
		return err
	}

	if len(msgRes.Messages) == 0 {
		log.Warn().Ints("messageIDs", messageIDs).Int("offset", offset).Str("typeOfPagination", typeOfPagination).Msg("no messages found in the database")
		return nil
	}

	var oldMsgs []*telego.Message
	filesLen := 0
	for _, msg := range msgRes.Messages {
		switch {
		case !user.Settings.ShowMyDeleted && user.Id == msg.From.ID:
			log.Debug().Msg("skip due user settings (self)")
			continue
		case !user.Settings.ShowPartnerDeleted && user.Id != msg.From.ID:
			log.Debug().Msg("skip due user settings (partner)")
			continue
		}

		media := utils.GetFile(msg)
		if media != nil {
			filesLen++
		}

		oldMsgs = append(oldMsgs, msg)
	}
	if len(oldMsgs) == 0 {
		log.Warn().Msg("no messages found after filter by user settings")
		return nil
	}

	if !itsCallbackQuery {
		correctMessagesLen, correctFilesLen = uint8(len(oldMsgs)), uint8(filesLen)
		dataID, err = h.repo.Mongo.SetDataDeleted(c, user.Id, messageIDs, correctMessagesLen, correctFilesLen)
		if err != nil {
			return err
		}

		if len(oldMsgs) > consts.MAX_BUTTONS {
			oldMsgs = oldMsgs[:consts.MAX_BUTTONS]
		}
	}

	rows := [][]telego.InlineKeyboardButton{}
	data := callbacks.HandleDeletedLogData{
		DataID: dataID,
		ChatID: chatID,
		Offset: offset,
	}
	rows = append(rows,
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(
				loc.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "business.deleted.fullMessages",
				}),
			).WithCallbackData(data.ToString()),
		),
	)

	if correctFilesLen != 0 {
		callbackData := callbacks.HandleDeletedFilesData{
			DataID: dataID,
			ChatID: chatID,
			Type:   callbacks.HandleDeletedFilesDataTypeData,
		}
		rows = append(rows,
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton(
					loc.MustLocalize(&i18n.LocalizeConfig{
						MessageID: "business.deleted.request.files",
						TemplateData: map[string]int{
							"Count": int(correctFilesLen),
						},
						PluralCount: int(correctFilesLen),
					}),
				).WithCallbackData(callbackData.ToString()),
			),
		)
	}

	if len(oldMsgs) > 1 {
		for i := 0; i < len(oldMsgs); i += 2 {
			row := make([]telego.InlineKeyboardButton, 0, 2)
			data := callbacks.HandleDeletedMessageData{
				MessageID:  oldMsgs[i].MessageID,
				ChatID:     chatID,
				DataID:     dataID,
				BackOffset: offset,
			}

			row = append(row, tu.InlineKeyboardButton(
				loc.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "business.deleted.messageItem",
					TemplateData: map[string]int{
						"Count": i + 1 + offset,
					},
				}),
			).WithCallbackData(data.ToString(callbacks.HandleDeletedMessageDataTypeMessage)))

			if i+1 < len(oldMsgs) {
				data.MessageID = oldMsgs[i+1].MessageID
				row = append(row, tu.InlineKeyboardButton(
					loc.MustLocalize(&i18n.LocalizeConfig{
						MessageID: "business.deleted.messageItem",
						TemplateData: map[string]int{
							"Count": i + 2 + offset,
						},
					}),
				).WithCallbackData(data.ToString(callbacks.HandleDeletedMessageDataTypeMessage)))
			}

			rows = append(rows, row)
		}

		if len(oldMsgs) > consts.MAX_BUTTONS || msgRes.Pagination.Backward || msgRes.Pagination.Forward {
			row := make([]telego.InlineKeyboardButton, 0, 2)

			paginationData := callbacks.HandleDeletedPaginationData{
				DataID: dataID,
				ChatID: chatID,
				Offset: offset,
			}

			if msgRes.Pagination.Backward {
				paginationData.TypeOfPagination = "b"
				row = append(
					row,
					tu.InlineKeyboardButton(
						loc.MustLocalize(&i18n.LocalizeConfig{
							MessageID: "arrow.backward",
						}),
					).
						WithCallbackData(paginationData.ToString()),
				)
			}
			if msgRes.Pagination.Forward {
				paginationData.TypeOfPagination = "f"
				row = append(
					row,
					tu.InlineKeyboardButton(
						loc.MustLocalize(&i18n.LocalizeConfig{
							MessageID: "arrow.forward",
						}),
					).
						WithCallbackData(paginationData.ToString()),
				)
			}

			rows = append(rows, row)
		}
	}

	var name string
	if itsCallbackQuery {
		name = format.Name(
			oldMsgs[0].Chat.FirstName,
			oldMsgs[0].Chat.LastName,
		)
	} else {
		name = format.Name(
			update.DeletedBusinessMessages.Chat.FirstName,
			update.DeletedBusinessMessages.Chat.LastName,
		)
	}
	summaryText := lFormat.SummarizeDeletedMessages(oldMsgs, name, oldMsgs[0].Chat.ID, loc, true, offset, int(correctMessagesLen))
	summaryText = format.CustomTruncateText(
		summaryText,
		consts.MAX_LEN,
		loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "business.deleted.overflowDescription",
		}),
		false,
	)

	if itsCallbackQuery {
		_, err = c.Bot().EditMessageText(c, tu.EditMessageText(
			tu.ID(user.Id),
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
		tu.ID(user.Id),
		summaryText,
	).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(tu.InlineKeyboard(rows...)),
	)

	return err
}

func (h *Handler) HandleEdited(c *th.Context, update telego.Update) error {
	message := update.EditedBusinessMessage
	loc := c.Value("loc").(*i18n.Localizer)
	user := c.Value("user").(*mongoRepository.User)
	log := c.Value("log").(*zerolog.Logger)

	oldMsg, err := h.repo.Mongo.GetMessage(
		context.Background(),
		&mongoRepository.GetMessageOptions{
			UserID:    user.Id,
			PeerID:    message.Chat.ID,
			MessageID: message.MessageID,
		},
	)
	if err != nil {
		log.Error().Err(err).
			Int("message_id", message.MessageID).
			Msg("failed GetMessage")
		errSave := h.repo.Mongo.SaveMessage(context.Background(), message, user.Id)
		if errSave != nil {
			log.Error().Err(errSave).Msg("error saving edited business message after failing to retrieve old message")
		}
		return err
	}

	switch {
	case !user.Settings.ShowMyEdits && user.Id == oldMsg.From.ID:
		log.Debug().Msg("skip due user settings (self)")
		return nil
	case !user.Settings.ShowPartnerEdits && user.Id != oldMsg.From.ID:
		log.Debug().Msg("skip due user settings (partner)")
		return nil
	}

	changes, mediaDiff := lFormat.EditedDiff(oldMsg, message, loc, true)

	if len(changes) == 0 {
		err = h.repo.Mongo.SaveMessage(context.Background(), message, user.Id)
		if err != nil {
			log.Error().Err(err).
				Int("message_id", message.MessageID).
				Msg("error saving edited business message (no changes detected)")
		}
		return err
	}

	name := format.Name(
		message.Chat.FirstName,
		message.Chat.LastName,
	)

	var (
		date       int64
		dateIsEdit bool
	)

	if oldMsg.EditDate == 0 {
		date = oldMsg.Date
	} else {
		date, dateIsEdit = oldMsg.EditDate, true
	}

	dataID, err := h.repo.Mongo.SetDataEdited(context.TODO(), &mongoRepository.SetDataEditedOptions{
		MessageID:     message.MessageID,
		UserID:        user.Id,
		OldDate:       date,
		OldDateIsEdit: dateIsEdit,
		NewDate:       message.EditDate,
	})
	if err != nil {
		return err
	}

	callbackData := callbacks.HandleEditedData{
		DataID: dataID,
		ChatID: message.Chat.ID,
	}
	replyMarkup := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(
				loc.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "business.edited.buttons.log",
				}),
			).WithCallbackData(callbackData.ToString(callbacks.HandleEditedDataTypeLog)),
		),
	)

	if mediaDiff.Removed != nil {
		replyMarkup.InlineKeyboard = append(replyMarkup.InlineKeyboard,
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton(
					loc.MustLocalize(&i18n.LocalizeConfig{
						MessageID: "business.edited.buttons.getFile",
					}),
				).WithCallbackData(callbackData.ToString(callbacks.HandleEditedDataTypeFiles)),
			),
		)
	}

	diffText := strings.Join(changes, "\n\n")
	resultText := loc.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "business.edited.message",
		TemplateData: map[string]any{
			"ChatID":           message.Chat.ID,
			"Diff":             diffText,
			"ResolvedChatName": name,
		},
	})
	if len(resultText) <= consts.MAX_LEN {
		resultText = loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "business.edited.messageOverflow",
			TemplateData: map[string]any{
				"ChatID":           message.Chat.ID,
				"ResolvedChatName": name,
			},
		})
	}
	_, err = c.Bot().SendMessage(c,
		tu.Message(
			tu.ID(user.Id),
			resultText,
		).
			WithParseMode(telego.ModeHTML).
			WithReplyMarkup(replyMarkup),
	)

	errSave := h.repo.Mongo.SaveMessage(context.Background(), message, user.Id)
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

func (h *Handler) HandleConnection(c *th.Context, update telego.Update) error {
	connection := update.BusinessConnection
	loc := c.Value("loc").(*i18n.Localizer)
	botID := c.Value("botID").(int64)

	isUpdated, err := h.repo.Mongo.UpdateBotUserConnection(c, connection, botID)
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

	if !isUpdated {
		_, err = c.Bot().SendMessage(c, tu.Message(
			tu.ID(connection.User.ID),
			text,
		).WithParseMode(telego.ModeHTML))
		return err
	}
	return nil
}
