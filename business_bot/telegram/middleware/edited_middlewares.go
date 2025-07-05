package middleware

import (
	"context"
	"fmt"
	"ssuspy-bot/repository"
	"ssuspy-bot/telegram/callbacks"
	"ssuspy-bot/telegram/utils"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/rs/zerolog/log"
)

func (h *MiddlewareGroup) EditedGetMessages(c *th.Context, update telego.Update) error {
	query := update.CallbackQuery
	loc := c.Value("loc").(*i18n.Localizer)
	iUser := c.Value("iUser").(*repository.IUser)

	data, err := callbacks.NewHandleEditedLogDataFromString(query.Data)
	if err != nil {
		log.Warn().Err(err).Str("data", query.Data).Msg("invalid callback data")
		utils.OnDataError(c, query.ID, loc)
		return fmt.Errorf("invalid callback data")
	}

	result, err := h.service.GetDataEdited(context.Background(), update.CallbackQuery.From.ID, data.DataID)
	if err != nil {
		log.Error().Err(err).Int64("dataID", data.DataID).Msg("error GetDataFullDeletedLogByUUID")
		utils.OnDataError(c, query.ID, loc)
		return err
	}

	msgs, _, err := h.service.GetMessages(
		context.Background(),
		&repository.GetMessagesOptions{
			ChatID:        data.ChatID,
			MessageIDs:    []int{result.MessageID},
			ConnectionIDs: iUser.BotUser.GetUserCurrentConnectionIDs(),
			WithEdits:     true,
		},
	)
	if err != nil {
		log.Error().Err(err).Int64("userID", iUser.User.ID).Msg("Error GetMessages for edited log")
		utils.OnDataError(c, query.ID, loc)
		return err
	}

	if len(msgs) < 2 {
		utils.OnDataError(c, query.ID, loc)
		return c.Bot().AnswerCallbackQuery(c, tu.CallbackQuery(query.ID))
	}

	var (
		newMsg *telego.Message
		oldMsg *telego.Message
	)

	for _, msg := range msgs {
		if oldMsg == nil {
			if (result.OldDateIsEdit && msg.EditDate == result.OldDate) ||
				(!result.OldDateIsEdit && msg.Date == result.OldDate && msg.EditDate == 0) {
				oldMsg = msg
			}
		}

		if newMsg == nil && msg.EditDate == result.NewDate {
			newMsg = msg
		}

		if oldMsg != nil && newMsg != nil {
			break
		}
	}

	if oldMsg == nil || newMsg == nil {
		utils.OnDataError(c, query.ID, loc)
	}

	c = c.WithValue("chatID", data.ChatID)
	c = c.WithValue("allEditedMessages", msgs)
	c = c.WithValue("editedMessage", newMsg)
	c = c.WithValue("oldEditedMessage", oldMsg)

	return c.Next(update)
}
