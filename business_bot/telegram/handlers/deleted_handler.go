package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/rs/zerolog"

	"ssuspy-bot/consts"
	"ssuspy-bot/telegram/callbacks"
	lFormat "ssuspy-bot/telegram/format"
	"ssuspy-bot/telegram/keyboard"
	sendMedia "ssuspy-bot/telegram/service/send_media"
	lUtils "ssuspy-bot/telegram/utils"
	"ssuspy-bot/types"
	"ssuspy-common/repository/mongoRepository"
	"ssuspy-common/telegram/format"
	"ssuspy-common/telegram/utils"
)

func (h *Handler) HandleDeletedLog(c *th.Context, update telego.Update) error {
	query := update.CallbackQuery
	loc := c.Value("loc").(*i18n.Localizer)
	user := c.Value("user").(*mongoRepository.User)
	log := c.Value("log").(*zerolog.Logger)

	data, err := callbacks.NewHandleDeletedLogDataFromString(query.Data)
	if err != nil {
		log.Warn().Err(err).Str("data", query.Data).Msg("invalid callback data")
		lUtils.OnDataError(c, query.ID, loc)
		return fmt.Errorf("invalid callback data")
	}

	result, err := h.repository.Mongo.GetDataDeleted(context.Background(), user.ID, data.DataID)
	if err != nil {
		log.Error().Err(err).Str("dataID", data.DataID.Hex()).Msg("error GetDataFullDeletedLogByUUID")
		lUtils.OnDataError(c, query.ID, loc)
		return err
	}

	msgRes, err := h.repository.Mongo.GetMessages(
		context.Background(),
		&mongoRepository.GetMessagesOptions{
			MessageIDs: result.MessageIDs,
			WithEdits:  true,
			UserID:     user.ID,
			PeerID:     data.ChatID,
		},
	)
	if err != nil {
		log.Warn().Err(err).Msg("failed GetMessages")
		lUtils.OnDataError(c, query.ID, loc)
		return err
	}

	if len(msgRes.Messages) == 0 {
		lUtils.OnDataError(c, query.ID, loc)
		return c.Bot().AnswerCallbackQuery(c, tu.CallbackQuery(query.ID))
	}

	name := format.Name(msgRes.Messages[0].Chat.FirstName, msgRes.Messages[0].Chat.LastName)

	now := time.Now().Format(consts.DATETIME_FOR_FILES)
	summaryText := lFormat.SummarizeDeletedMessages(msgRes.Messages, name, loc, false, data.Offset, len(msgRes.Messages))
	files := []telego.InputMedia{
		tu.MediaDocument(lFormat.GetMDInputFile(summaryText, fmt.Sprintf("%d-summary-%s", data.ChatID, now))),
	}

	filteredMsgs := lFormat.FilterMessagesByDate(msgRes.Messages)
	withEdits := len(filteredMsgs) != len(msgRes.Messages)
	if withEdits {
		jsonBytesWithEdits, _ := json.MarshalIndent(msgRes.Messages, "", "  ")
		files = append(
			files,
			tu.MediaDocument(tu.FileFromBytes(jsonBytesWithEdits, fmt.Sprintf("%d-all-json-%s.json", data.ChatID, now))),
		)
	}

	jsonBytes, _ := json.MarshalIndent(filteredMsgs, "", "  ")
	files = append(
		files,
		tu.MediaDocument(tu.FileFromBytes(jsonBytes, fmt.Sprintf("%d-json-%s.json", data.ChatID, now))).
			WithCaption(
				loc.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "business.deleted.request.message",
					TemplateData: map[string]bool{
						"WithEdits": withEdits,
					},
				}),
			).WithParseMode(telego.ModeHTML),
	)

	if err := sendMedia.SendMediaInGroups(c.Bot(), c, user.ID, files, query.Message.GetMessageID()); err != nil {
		log.Warn().Err(err).Msg("Error sending media to user")
		lUtils.OnFilesError(c, user.ID, loc, query.Message.GetMessageID())
	}

	return c.Bot().AnswerCallbackQuery(c, tu.CallbackQuery(query.ID))
}

func (h *Handler) HandleDeletedMessage(c *th.Context, update telego.Update) error {
	query := update.CallbackQuery
	loc := c.Value("loc").(*i18n.Localizer)
	user := c.Value("user").(*mongoRepository.User)
	log := c.Value("log").(*zerolog.Logger)

	data, err := callbacks.NewHandleDeletedMessageDataFromString(query.Data)
	if err != nil {
		log.Warn().Err(err).Str("data", query.Data).Msg("invalid callback data")
		lUtils.OnDataError(c, query.ID, loc)
		return fmt.Errorf("invalid callback data")
	}

	msg, err := h.repository.Mongo.GetMessage(
		context.Background(), &mongoRepository.GetMessageOptions{
			UserID:    user.ID,
			PeerID:    data.ChatID,
			MessageID: data.MessageID,
		},
	)
	if err != nil {
		log.Error().
			Err(err).
			Int("messageID", data.MessageID).
			Msg("error GetMessage for full deleted message")
		lUtils.OnDataError(c, query.ID, loc)
		return err
	}

	buttons := [][]telego.InlineKeyboardButton{
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(
				loc.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "business.deleted.request.oneDetails",
				}),
			).WithCallbackData(data.ToString(callbacks.HandleDeletedMessageDataTypeDetails)),
		),
	}

	file := utils.GetFile(msg)
	if file != nil {
		callbackData := callbacks.HandleDeletedFilesData{
			MessageID: data.MessageID,
			ChatID:    data.ChatID,
			Type:      callbacks.HandleDeletedFilesDataTypeMessage,
		}
		buttons = append(buttons, tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(
				loc.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "business.deleted.request.file",
				}),
			).WithCallbackData(callbackData.ToString()),
		))
	}

	callbackData := callbacks.HandleDeletedPaginationData{
		DataID: data.DataID,
		ChatID: data.ChatID,
		Offset: data.BackOffset,
	}

	buttons = append(buttons, tu.InlineKeyboardRow(
		keyboard.BuildBackButton(loc, callbackData.ToString()),
	))

	summaryText := lFormat.SummarizeDeletedMessage(msg, loc, true)
	if _, err := c.Bot().EditMessageText(c, tu.EditMessageText(tu.ID(query.From.ID), query.Message.GetMessageID(), summaryText).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(tu.InlineKeyboard(buttons...)),
	); err != nil {
		log.Warn().Err(err).Msg("Error sending deleted message summary")
		return err
	}

	return c.Bot().AnswerCallbackQuery(c, tu.CallbackQuery(query.ID))
}

func (h *Handler) HandleDeletedMessageDetails(c *th.Context, update telego.Update) error {
	query := update.CallbackQuery
	loc := c.Value("loc").(*i18n.Localizer)
	user := c.Value("user").(*mongoRepository.User)
	log := c.Value("log").(*zerolog.Logger)

	data, err := callbacks.NewHandleDeletedMessageDataFromString(query.Data)
	if err != nil {
		log.Warn().Err(err).Str("data", query.Data).Msg("invalid callback data")
		lUtils.OnDataError(c, query.ID, loc)
		return fmt.Errorf("invalid callback data")
	}

	msgRes, err := h.repository.Mongo.GetMessages(
		context.Background(),
		&mongoRepository.GetMessagesOptions{
			UserID:     user.ID,
			PeerID:     data.ChatID,
			MessageIDs: []int{data.MessageID},
			WithEdits:  true,
		},
	)
	if err != nil || len(msgRes.Messages) == 0 {
		log.Warn().Err(err).Int("messageID", data.MessageID).Msg("failed GetMessages")
		lUtils.OnDataError(c, query.ID, loc)
		if err == nil {
			err = fmt.Errorf("no messages found for details")
		}
		return err
	}

	name := format.Name(msgRes.Messages[0].Chat.FirstName, msgRes.Messages[0].Chat.LastName)

	now := time.Now().Format(consts.DATETIME_FOR_FILES)
	summaryText := lFormat.SummarizeDeletedMessages(msgRes.Messages, name, loc, false, data.BackOffset, len(msgRes.Messages))
	files := []telego.InputMedia{
		tu.MediaDocument(lFormat.GetMDInputFile(summaryText, fmt.Sprintf("msg-%d-summary-%s", data.MessageID, now))),
	}
	if len(msgRes.Messages) >= 2 {
		jsonBytesAllEdits, _ := json.MarshalIndent(msgRes.Messages, "", "  ")
		files = append(files, tu.MediaDocument(tu.FileFromBytes(jsonBytesAllEdits, fmt.Sprintf("msg-%d-all-json-%s.json", data.MessageID, now))))
	}

	latestMsg := lFormat.FilterMessagesByDate(msgRes.Messages)
	jsonBytesLatest, _ := json.MarshalIndent(latestMsg, "", "  ")
	files = append(
		files,
		tu.MediaDocument(tu.FileFromBytes(jsonBytesLatest, fmt.Sprintf("msg-%d-latest-%s.json", data.MessageID, now))).
			WithCaption(
				loc.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "business.deleted.request.message",
					TemplateData: map[string]bool{
						"WithEdits": len(msgRes.Messages) >= 2,
					},
				}),
			).WithParseMode(telego.ModeHTML),
	)

	if err := sendMedia.SendMediaInGroups(c.Bot(), c, user.ID, files, query.Message.GetMessageID()); err != nil {
		log.Warn().Err(err).Msg("Error sending media to user")
		lUtils.OnFilesError(c, user.ID, loc, query.Message.GetMessageID())
	}
	return c.Bot().AnswerCallbackQuery(c, tu.CallbackQuery(query.ID))
}

func (h *Handler) HandleGetDeletedFiles(c *th.Context, update telego.Update) error {
	query := update.CallbackQuery
	user := c.Value("user").(*mongoRepository.User)
	loc := c.Value("loc").(*i18n.Localizer)
	log := c.Value("log").(*zerolog.Logger)

	data, err := callbacks.NewHandleDeletedFilesFromString(query.Data)
	if err != nil {
		log.Warn().Err(err).Str("data", query.Data).Msg("invalid callback data")
		lUtils.OnDataError(c, query.ID, loc)
		return fmt.Errorf("invalid callback data")
	}

	var messageIDs []int
	switch data.Type {
	case callbacks.HandleDeletedFilesDataTypeMessage:
		messageIDs = []int{data.MessageID}
	case callbacks.HandleDeletedFilesDataTypeData:
		callbackData, err := h.repository.Mongo.GetDataDeleted(context.Background(), user.ID, data.DataID)
		if err != nil {
			log.Warn().Str("dataID", data.DataID.Hex()).Err(err).Msg("failed GetDataDeleted")
			lUtils.OnFilesError(c, user.ID, loc, query.Message.GetMessageID())
			return err
		}

		messageIDs = callbackData.MessageIDs
	}

	msgRes, err := h.repository.Mongo.GetMessages(
		context.Background(),
		&mongoRepository.GetMessagesOptions{
			UserID:     user.ID,
			PeerID:     data.ChatID,
			MessageIDs: messageIDs,
		},
	)
	if err != nil || len(msgRes.Messages) == 0 {
		log.Warn().Err(err).Int64("userID", user.ID).Msg("Error GetMessages for get deleted files log")
		lUtils.OnDataError(c, query.ID, loc)
		if err == nil {
			err = fmt.Errorf("no messages found for get deleted files log")
		}
		return err
	}

	var files []*types.MediaItemProcess
	processedMediaGroupIDs := make(map[string]bool)

	for _, msg := range msgRes.Messages {
		if msg.MediaGroupID != "" {
			if processedMediaGroupIDs[msg.MediaGroupID] {
				continue
			}
			currentMediaGroupItems := []*telego.Message{}
			for _, m := range msgRes.Messages {
				if m.MediaGroupID == msg.MediaGroupID {
					currentMediaGroupItems = append(currentMediaGroupItems, m)
				}
			}

			groupCaption := ""
			if len(currentMediaGroupItems) > 0 {
				groupCaption = currentMediaGroupItems[0].Caption
				if groupCaption == "" {
					for _, mGroupItem := range currentMediaGroupItems {
						if mGroupItem.Caption != "" {
							groupCaption = mGroupItem.Caption
							break
						}
					}
				}
			}

			for i, groupMsg := range currentMediaGroupItems {
				mediaFile := utils.GetFile(groupMsg)
				if mediaFile == nil {
					continue
				}

				caption := ""
				if i == 0 && groupCaption != "" {
					caption = loc.MustLocalize(&i18n.LocalizeConfig{
						MessageID: "sendMediaInGroups",
						TemplateData: map[string]string{
							"Text": format.Caption(groupCaption),
						},
					})
				}

				files = append(files, &types.MediaItemProcess{
					Type:     mediaFile.Type,
					FileID:   mediaFile.FileID,
					FileSize: mediaFile.FileSize,
					Caption:  caption,
				})
			}
			processedMediaGroupIDs[msg.MediaGroupID] = true
		} else {
			mediaFile := utils.GetFile(msg)
			if mediaFile == nil {
				continue
			}

			files = append(files, &types.MediaItemProcess{
				Type:     mediaFile.Type,
				FileID:   mediaFile.FileID,
				FileSize: mediaFile.FileSize,
			})
		}
	}

	if len(files) == 0 {
		return lUtils.OnDataError(c, query.ID, loc)
	} else {
		sort := lUtils.SortFiles(files)
		converted := lUtils.ConvertFileInfosGroupsToInputMediaGroups(sort)
		for i, sortFiles := range converted {
			if err = sendMedia.SendMediaInGroups(c.Bot(), c, user.ID, sortFiles, query.Message.GetMessageID()); err != nil {
				log.Warn().Err(err).Int("batchIndex", i).Msg("failed sending files for get deleted files")
				lUtils.OnFilesError(c, user.ID, loc, query.Message.GetMessageID())
			}
		}
	}

	return nil
}
