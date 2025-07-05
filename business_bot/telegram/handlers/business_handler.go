package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"ssuspy-bot/consts"
	"ssuspy-bot/redis"
	"ssuspy-bot/repository"
	"ssuspy-bot/telegram/callbacks"
	"ssuspy-bot/telegram/format"
	"ssuspy-bot/telegram/utils"
	"ssuspy-bot/types"
)

func (h *Handler) HandleMessage(c *th.Context, update telego.Update) error {
	message := update.BusinessMessage

	message.Text = format.TruncateText(message.Text, consts.MAX_USER_MESSAGE_TEXT_LEN, false)
	message.Caption = format.TruncateText(message.Caption, consts.MAX_USER_MESSAGE_TEXT_LEN, false)

	err := h.service.SaveMessage(context.Background(), message)
	if err != nil {
		log.Warn().
			Err(err).
			Int64("chatID", message.Chat.ID).
			Int("messageID", message.MessageID).
			Msg("error saving message")
		return nil
	}

	name := format.Name(
		message.Chat.FirstName,
		message.Chat.LastName,
	)

	err = h.service.UpdateChatName(
		c,
		message.Chat.ID,
		name,
	)
	if err != nil {
		log.Warn().Err(err).Msg("failed save/update chat name")
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
	iUser := c.Value("iUser").(*repository.IUser)

	file := utils.GetFile(replyToMessage)
	if file == nil {
		return nil
	}

	fileExists, err := h.service.CreateFileIfNotExists(c, file.FileID, iUser.User.ID, message.Chat.ID)
	if err != nil || !fileExists {
		return err
	}

	protectedMessage, err := c.Bot().SendMessage(c, tu.Message(
		tu.ID(iUser.User.ID),
		loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "business.restrictedMedia",
		}),
	).WithParseMode(telego.ModeHTML))
	if err != nil {
		return err
	}

	err = h.rdb.EnqueueJob(c, consts.REDIS_QUEUE_FILES, redis.Job{
		File:             file,
		UserID:           iUser.User.ID,
		ChatID:           message.Chat.ID,
		MessageID:        protectedMessage.MessageID,
		Caption:          replyToMessage.Caption,
		UserLanguageCode: iUser.User.LanguageCode,
		BotID:            botID,
	})
	if err != nil {
		return err
	}

	return err
}

func (h *Handler) HandleDeleted(c *th.Context, update telego.Update) error {
	loc := c.Value("loc").(*i18n.Localizer)
	iUser := c.Value("iUser").(*repository.IUser)
	log := c.Value("log").(*zerolog.Logger)

	itsCallbackQuery := c.Value("itsCallbackQuery").(bool)
	chatID := c.Value("chatID").(int64)
	messageIDs := c.Value("messageIDs").([]int)

	var (
		limit              int
		offset             int
		typeOfPagination   string
		dataID             int64
		correctMessagesLen uint8
		correctFilesLen    uint8
	)
	if itsCallbackQuery {
		data, err := callbacks.NewHandleDeletedPaginationDataFromString(update.CallbackQuery.Data)
		if err != nil {
			log.Warn().Err(err).Str("data", update.CallbackQuery.Data).Msg("invalid callback data")
			return fmt.Errorf("invalid callback data")
		}

		result, err := h.service.GetDataDeleted(context.Background(), update.CallbackQuery.From.ID, data.DataID)
		if err != nil {
			log.Error().Err(err).Int64("dataID", data.DataID).Msg("error GetDataFullDeletedLogByUUID")

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

	unfilteredOldMsgs, pagination, err := h.service.GetMessages(
		context.Background(),
		&repository.GetMessagesOptions{
			ChatID:        chatID,
			MessageIDs:    messageIDs,
			ConnectionIDs: iUser.BotUser.GetUserCurrentConnectionIDs(),
			Limit:         limit,
			Offset:        offset,
		},
	)
	if err != nil {
		return err
	}

	if len(unfilteredOldMsgs) == 0 {
		log.Warn().Ints("messageIDs", messageIDs).Int("offset", offset).Str("typeOfPagination", typeOfPagination).Msg("no messages found in the database")
		return nil
	}

	var oldMsgs []*telego.Message
	filesLen := 0
	for _, msg := range unfilteredOldMsgs {
		switch {
		case !iUser.User.Settings.ShowMyDeleted && iUser.User.ID == msg.From.ID:
			log.Debug().Msg("skip due user settings (self)")
			continue
		case !iUser.User.Settings.ShowPartnerDeleted && iUser.User.ID != msg.From.ID:
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
		dataID, err = h.service.SetDataDeleted(context.TODO(), iUser.User.ID, messageIDs, correctMessagesLen, correctFilesLen)
		if err != nil {
			return err
		}

		if len(oldMsgs) > consts.MAX_BUTTONS {
			oldMsgs = oldMsgs[:consts.MAX_BUTTONS]
		}

		pagination.Forward = true
	}

	rows := [][]telego.InlineKeyboardButton{}
	if len(oldMsgs) > 0 {
		data := types.HandleDeletedLogData{
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
			callbackData := types.HandleDeletedFilesData{
				DataID: dataID,
				ChatID: chatID,
				Type:   types.HandleDeletedFilesDataTypeData,
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
				data := types.HandleDeletedMessageData{
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
				).WithCallbackData(data.ToString(types.HandleDeletedMessageDataTypeMessage)))

				if i+1 < len(oldMsgs) {
					data.MessageID = oldMsgs[i+1].MessageID
					row = append(row, tu.InlineKeyboardButton(
						loc.MustLocalize(&i18n.LocalizeConfig{
							MessageID: "business.deleted.messageItem",
							TemplateData: map[string]int{
								"Count": i + 2 + offset,
							},
						}),
					).WithCallbackData(data.ToString(types.HandleDeletedMessageDataTypeMessage)))
				}

				rows = append(rows, row)
			}

			if len(oldMsgs) > consts.MAX_BUTTONS || pagination.Backward || pagination.Forward {
				row := make([]telego.InlineKeyboardButton, 0, 2)

				paginationData := types.HandleDeletedPaginationData{
					DataID: dataID,
					ChatID: chatID,
					Offset: offset,
				}

				if pagination.Backward {
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
				if pagination.Forward {
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
	}

	var name string
	if itsCallbackQuery {
		chatResolve, err := h.service.FindChatName(c, chatID)
		if err != nil {
			name = strconv.FormatInt(chatID, 10)
		} else {
			name = chatResolve.Name
		}
	} else {
		name = format.Name(
			update.DeletedBusinessMessages.Chat.FirstName,
			update.DeletedBusinessMessages.Chat.LastName,
		)
	}
	summaryText := format.SummarizeDeletedMessages(oldMsgs, name, loc, true, offset, int(correctMessagesLen))
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
			tu.ID(iUser.User.ID),
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
		tu.ID(iUser.User.ID),
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
	iUser := c.Value("iUser").(*repository.IUser)
	log := c.Value("log").(*zerolog.Logger)

	oldMsg, err := h.service.GetMessage(
		context.Background(),
		&repository.GetMessageOptions{
			ChatID:        message.Chat.ID,
			MessageID:     message.MessageID,
			ConnectionIDs: iUser.BotUser.GetUserCurrentConnectionIDs(),
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

	switch {
	case !iUser.User.Settings.ShowMyEdits && iUser.User.ID == oldMsg.From.ID:
		log.Debug().Msg("skip due user settings (self)")
		return nil
	case !iUser.User.Settings.ShowPartnerEdits && iUser.User.ID != oldMsg.From.ID:
		log.Debug().Msg("skip due user settings (partner)")
		return nil
	}

	changes, mediaDiff := format.EditedDiff(oldMsg, message, loc, true)

	if len(changes) == 0 {
		err = h.service.SaveMessage(context.Background(), message)
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

	diffText := strings.Join(changes, "\n\n")
	editedAt := time.Unix(int64(message.EditDate), 0).Format(consts.DATETIME_FOR_MESSAGE)
	formattedText := loc.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "business.edited.message",
		TemplateData: map[string]any{
			"ChatID":           message.Chat.ID,
			"Date":             editedAt,
			"Diff":             diffText,
			"ResolvedChatName": name,
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
		UserID:        iUser.User.ID,
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
				tu.ID(iUser.User.ID),
				formattedText,
			).WithParseMode(telego.ModeHTML).WithReplyMarkup(replyMarkup),
		)
	} else {
		_, err = c.Bot().SendMessage(c, tu.Message(
			tu.ID(iUser.User.ID),
			loc.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "business.edited.messageOverflow",
				TemplateData: map[string]any{
					"ChatID":           message.Chat.ID,
					"Date":             editedAt,
					"ResolvedChatName": name,
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

func (h *Handler) HandleConnection(c *th.Context, update telego.Update) error {
	connection := update.BusinessConnection
	loc := c.Value("loc").(*i18n.Localizer)
	botID := c.Value("botID").(int64)

	isUpdated, err := h.service.UpdateBotUserConnection(context.Background(), connection, botID)
	if err != nil {
		return err
	}

	var text string
	if connection.IsEnabled {
		name := format.Name(connection.User.FirstName, connection.User.LastName)

		err = h.service.UpdateBotUserSendMessages(c, connection.User.ID, botID, true)
		if err != nil {
			return err
		}

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
