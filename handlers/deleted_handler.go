package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"ssuspy-bot/callbacks"
	"ssuspy-bot/consts"
	"ssuspy-bot/format"
	"ssuspy-bot/repository"
	"ssuspy-bot/types"
	"ssuspy-bot/utils"
)

func (h *Handler) HandleDeletedPagination(c *th.Context, query telego.CallbackQuery) error {
	loc := c.Value("loc").(*i18n.Localizer)
	user := c.Value("user").(*repository.User)
	log := c.Value("log").(*zerolog.Logger)

	data, err := callbacks.NewHandleDeletedPaginationDataFromString(query.Data)
	if err != nil {
		log.Warn().Err(err).Str("data", query.Data).Msg("invalid callback data")
		utils.OnDataError(c, query.ID, loc)
		return fmt.Errorf("invalid callback data")
	}

	result, err := h.service.GetDataDeleted(context.Background(), user.ID, data.DataID)
	if err != nil {
		log.Warn().Err(err).Int64("dataID", data.ChatID).Msg("failed GetDataFullDeletedLogByUUID")
		utils.OnDataError(c, query.ID, loc)

		return err
	}

	var offset int
	if data.TypeOfPagination == "f" {
		offset = data.Offset + consts.MAX_BUTTONS
	} else if data.TypeOfPagination == "b" {
		offset = max(data.Offset-consts.MAX_BUTTONS, 0)
	}

	messageID := query.Message.GetMessageID()
	oldMsgs, pagination, err := h.service.GetMessages(
		context.Background(),
		&repository.GetMessagesOptions{
			ChatID:        data.ChatID,
			MessageIDs:    result.MessageIDs,
			ConnectionIDs: user.GetUserCurrentConnectionIDs(),
			Offset:        offset,
			Limit:         consts.MAX_BUTTONS,
		},
	)
	if err != nil {
		return err
	}

	if len(oldMsgs) == 0 {
		log.Warn().Int64("chatID", data.ChatID).Ints("messageIDs", result.MessageIDs).Msg("no messages found in the database")
		return nil
	}

	rows := utils.DeletedRows(data.ChatID, user, loc, oldMsgs, pagination, offset, data.DataID)

	if _, err := c.Bot().EditMessageReplyMarkup(c, tu.EditMessageReplayMarkup(
		tu.ID(user.ID),
		messageID,
		tu.InlineKeyboard(rows...),
	)); err != nil {
		log.Warn().Err(err).Msg("Error sending media to user")
		utils.OnFilesError(c, user.ID, loc, query.Message.GetMessageID())
		return err
	}

	return c.Bot().AnswerCallbackQuery(c, tu.CallbackQuery(query.ID))
}

func (h *Handler) HandleDeletedLog(c *th.Context, query telego.CallbackQuery) error {
	loc := c.Value("loc").(*i18n.Localizer)
	user := c.Value("user").(*repository.User)
	log := c.Value("log").(*zerolog.Logger)

	data, err := callbacks.NewHandleDeletedLogDataFromString(query.Data)
	if err != nil {
		log.Warn().Err(err).Str("data", query.Data).Msg("invalid callback data")
		utils.OnDataError(c, query.ID, loc)
		return fmt.Errorf("invalid callback data")
	}

	result, err := h.service.GetDataDeleted(context.Background(), user.ID, data.DataID)
	if err != nil {
		log.Error().Err(err).Int64("dataID", data.DataID).Msg("error GetDataFullDeletedLogByUUID")
		utils.OnDataError(c, query.ID, loc)
		return err
	}

	msgs, _, err := h.service.GetMessages(
		context.Background(),
		&repository.GetMessagesOptions{
			ChatID:        data.ChatID,
			MessageIDs:    result.MessageIDs,
			ConnectionIDs: user.GetUserCurrentConnectionIDs(),
			WithEdits:     true,
		},
	)
	if err != nil {
		log.Warn().Err(err).Msg("failed GetMessages")
		utils.OnDataError(c, query.ID, loc)
		return err
	}

	if len(msgs) == 0 {
		utils.OnDataError(c, query.ID, loc)
		return c.Bot().AnswerCallbackQuery(c, tu.CallbackQuery(query.ID))
	}

	var name string
	chatResolve, err := h.service.FindChatName(c, data.ChatID)
	if err != nil {
		name = strconv.FormatInt(data.ChatID, 10)
	} else {
		name = chatResolve.Name
	}

	now := time.Now().Format(consts.DATETIME_FOR_FILES)
	summaryText := format.SummarizeDeletedMessages(msgs, name, loc, false)
	files := []telego.InputMedia{
		tu.MediaDocument(format.GetMDInputFile(summaryText, fmt.Sprintf("%d-summary-%s", data.ChatID, now))),
	}

	filteredMsgs := format.FilterMessagesByDate(msgs)
	withEdits := len(filteredMsgs) != len(msgs)
	if withEdits {
		jsonBytesWithEdits, _ := json.MarshalIndent(msgs, "", "  ")
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

	if err := utils.SendMediaInGroups(c.Bot(), c, user.ID, files, query.Message.GetMessageID()); err != nil {
		log.Warn().Err(err).Msg("Error sending media to user")
		utils.OnFilesError(c, user.ID, loc, query.Message.GetMessageID())
	}

	return c.Bot().AnswerCallbackQuery(c, tu.CallbackQuery(query.ID))
}

func (h *Handler) HandleDeletedMessage(c *th.Context, query telego.CallbackQuery) error {
	loc := c.Value("loc").(*i18n.Localizer)
	user := c.Value("user").(*repository.User)
	log := c.Value("log").(*zerolog.Logger)

	data, err := callbacks.NewHandleDeletedMessageDataFromString(query.Data)
	if err != nil {
		log.Warn().Err(err).Str("data", query.Data).Msg("invalid callback data")
		utils.OnDataError(c, query.ID, loc)
		return fmt.Errorf("invalid callback data")
	}

	msg, err := h.service.GetMessage(
		context.Background(), &repository.GetMessageOptions{
			ChatID:        data.ChatID,
			MessageID:     data.MessageID,
			ConnectionIDs: user.GetUserCurrentConnectionIDs(),
		},
	)
	if err != nil {
		log.Error().
			Err(err).
			Int("messageID", data.MessageID).
			Msg("error GetMessage for full deleted message")
		utils.OnDataError(c, query.ID, loc)
		return err
	}

	buttons := [][]telego.InlineKeyboardButton{
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(
				loc.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "business.deleted.request.oneDetails",
				}),
			).WithCallbackData(data.ToString(types.HandleDeletedMessageDataTypeDetails)),
		),
	}

	file := utils.GetFile(msg)
	if file != nil {
		callbackData := types.HandleDeletedFilesData{
			MessageID: data.MessageID,
			ChatID:    data.ChatID,
			Type:      types.HandleDeletedFilesDataTypeMessage,
		}
		buttons = append(buttons, tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(
				loc.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "business.deleted.request.file",
				}),
			).WithCallbackData(callbackData.ToString()),
		))
	}

	callbackData := types.HandleBusinessData{
		DataID: data.DataID,
		ChatID: data.ChatID,
	}
	buttons = append(buttons, tu.InlineKeyboardRow(
		tu.InlineKeyboardButton(
			loc.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "back",
			}),
		).WithCallbackData(callbackData.ToString(types.HandleBusinessDataTypeDeleted)),
	))

	summaryText := format.SummarizeDeletedMessage(msg, loc, true)
	if _, err := c.Bot().EditMessageText(c, tu.EditMessageText(tu.ID(query.From.ID), query.Message.GetMessageID(), summaryText).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(tu.InlineKeyboard(buttons...)),
	); err != nil {
		log.Warn().Err(err).Msg("Error sending deleted message summary")
		return err
	}

	return c.Bot().AnswerCallbackQuery(c, tu.CallbackQuery(query.ID))
}

func (h *Handler) HandleDeletedMessageDetails(c *th.Context, query telego.CallbackQuery) error {
	loc := c.Value("loc").(*i18n.Localizer)
	user := c.Value("user").(*repository.User)
	log := c.Value("log").(*zerolog.Logger)

	data, err := callbacks.NewHandleDeletedMessageDataFromString(query.Data)
	if err != nil {
		log.Warn().Err(err).Str("data", query.Data).Msg("invalid callback data")
		utils.OnDataError(c, query.ID, loc)
		return fmt.Errorf("invalid callback data")
	}

	msgs, _, err := h.service.GetMessages(
		context.Background(),
		&repository.GetMessagesOptions{
			ChatID:        data.ChatID,
			MessageIDs:    []int{data.MessageID},
			ConnectionIDs: user.GetUserCurrentConnectionIDs(),
			WithEdits:     true,
		},
	)
	if err != nil || len(msgs) == 0 {
		log.Warn().Err(err).Int("messageID", data.MessageID).Msg("failed GetMessages")
		utils.OnDataError(c, query.ID, loc)
		if err == nil {
			err = fmt.Errorf("no messages found for details")
		}
		return err
	}

	var name string
	chatResolve, err := h.service.FindChatName(c, data.ChatID)
	if err != nil {
		name = strconv.FormatInt(data.ChatID, 10)
	} else {
		name = chatResolve.Name
	}

	now := time.Now().Format(consts.DATETIME_FOR_FILES)
	summaryText := format.SummarizeDeletedMessages(msgs, name, loc, false)
	files := []telego.InputMedia{
		tu.MediaDocument(format.GetMDInputFile(summaryText, fmt.Sprintf("msg-%d-summary-%s", data.MessageID, now))),
	}
	if len(msgs) > 1 {
		jsonBytesAllEdits, _ := json.MarshalIndent(msgs, "", "  ")
		files = append(files, tu.MediaDocument(tu.FileFromBytes(jsonBytesAllEdits, fmt.Sprintf("msg-%d-all-json-%s.json", data.MessageID, now))))
	}

	latestMsg := format.FilterMessagesByDate(msgs)
	jsonBytesLatest, _ := json.MarshalIndent(latestMsg, "", "  ")
	files = append(
		files,
		tu.MediaDocument(tu.FileFromBytes(jsonBytesLatest, fmt.Sprintf("msg-%d-latest-%s.json", data.MessageID, now))).
			WithCaption(
				loc.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "business.deleted.request.message",
					TemplateData: map[string]bool{
						"WithEdits": len(msgs) > 1,
					},
				}),
			).WithParseMode(telego.ModeHTML),
	)

	if err := utils.SendMediaInGroups(c.Bot(), c, user.ID, files, query.Message.GetMessageID()); err != nil {
		log.Warn().Err(err).Msg("Error sending media to user")
		utils.OnFilesError(c, user.ID, loc, query.Message.GetMessageID())
	}
	return c.Bot().AnswerCallbackQuery(c, tu.CallbackQuery(query.ID))
}

func (h *Handler) HandleGetDeletedFiles(c *th.Context, query telego.CallbackQuery) error {
	user := c.Value("user").(*repository.User)
	loc := c.Value("loc").(*i18n.Localizer)

	data, err := callbacks.NewHandleDeletedFilesFromString(query.Data)
	if err != nil {
		log.Warn().Err(err).Str("data", query.Data).Msg("invalid callback data")
		utils.OnDataError(c, query.ID, loc)
		return fmt.Errorf("invalid callback data")
	}

	var messageIDs []int
	switch data.Type {
	case types.HandleDeletedFilesDataTypeMessage:
		messageIDs = []int{data.MessageID}
	case types.HandleDeletedFilesDataTypeData:
		callbackData, err := h.service.GetDataDeleted(context.Background(), user.ID, data.DataID)
		if err != nil {
			log.Warn().Int64("dataID", data.DataID).Err(err).Msg("failed GetDataDeleted")
			utils.OnFilesError(c, user.ID, loc, query.Message.GetMessageID())
			return err
		}

		messageIDs = callbackData.MessageIDs
	}

	msgs, _, err := h.service.GetMessages(
		context.Background(),
		&repository.GetMessagesOptions{
			ChatID:        data.ChatID,
			MessageIDs:    messageIDs,
			ConnectionIDs: user.GetUserCurrentConnectionIDs(),
		},
	)
	if err != nil || len(msgs) == 0 {
		log.Warn().Err(err).Int64("userID", user.ID).Msg("Error GetMessages for get deleted files log")
		utils.OnDataError(c, query.ID, loc)
		if err == nil {
			err = fmt.Errorf("no messages found for get deleted files log")
		}
		return err
	}

	var files []*types.MediaItemProcess
	processedMediaGroupIDs := make(map[string]bool)

	for _, msg := range msgs {
		if msg.MediaGroupID != "" {
			if processedMediaGroupIDs[msg.MediaGroupID] {
				continue
			}
			currentMediaGroupItems := []*telego.Message{}
			for _, m := range msgs {
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
		utils.OnDataError(c, query.ID, loc)
	} else {
		sort := utils.SortFiles(files)
		converted := utils.ConvertFileInfosGroupsToInputMediaGroups(sort)
		for i, sortFiles := range converted {
			if err = utils.SendMediaInGroups(c.Bot(), c, user.ID, sortFiles, query.Message.GetMessageID()); err != nil {
				log.Warn().Err(err).Int("batchIndex", i).Msg("failed sending files for get deleted files")
			}
			if err != nil {
				utils.OnFilesError(c, user.ID, loc, query.Message.GetMessageID())
			}
		}
	}

	return c.Bot().AnswerCallbackQuery(c, tu.CallbackQuery(query.ID))
}
