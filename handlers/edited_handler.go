package handlers

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/rs/zerolog/log"

	"ssuspy-bot/consts"
	"ssuspy-bot/format"
	"ssuspy-bot/repository"
	"ssuspy-bot/utils"
)

func (h *Handler) HandleEditedLog(c *th.Context, query telego.CallbackQuery) error {
	loc := c.Value("loc").(*i18n.Localizer)
	user := c.Value("user").(*repository.User)

	chatID := c.Value("chatID").(int64)
	msgs := c.Value("allEditedMessages").([]*telego.Message)
	newMsg := c.Value("editedMessage").(*telego.Message)
	oldMsg := c.Value("oldEditedMessage").(*telego.Message)

	changes, _ := format.EditedDiff(oldMsg, newMsg, loc, false)
	if len(changes) == 0 {
		log.Warn().
			Int64("chat_id", newMsg.Chat.ID).
			Int("message_id", newMsg.MessageID).
			Msg("no data found for the specified messageID")
		return nil
	}

	result := map[string]any{
		"oldMessage": oldMsg,
		"newMessage": newMsg,
	}

	now := time.Now().Format(consts.DATETIME_FOR_FILES)
	diffText := strings.Join(changes, "\n\n")
	files := []telego.InputMedia{
		tu.MediaDocument(format.GetMDInputFile(diffText, fmt.Sprintf("%d-diff-%s", chatID, now))),
	}
	if len(msgs) > 2 {
		jsonBytesWithAll, _ := json.MarshalIndent(msgs, "", "  ")
		files = append(files, tu.MediaDocument(tu.FileFromBytes(jsonBytesWithAll, fmt.Sprintf("%d-all-json-%s.json", chatID, now))))
	}

	jsonBytes, _ := json.MarshalIndent(result, "", "  ")
	files = append(files, tu.MediaDocument(tu.FileFromBytes(jsonBytes, fmt.Sprintf("%d-json-%s.json", chatID, now))).
		WithCaption(
			loc.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "business.edited.request",
				TemplateData: map[string]bool{
					"WithEdits": len(msgs) > 2,
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
func (h *Handler) HandleEditedFiles(c *th.Context, query telego.CallbackQuery) error {
	user := c.Value("user").(*repository.User)
	loc := c.Value("loc").(*i18n.Localizer)

	newMsg := c.Value("editedMessage").(*telego.Message)
	oldMsg := c.Value("oldEditedMessage").(*telego.Message)

	newMedia := utils.GetFile(newMsg)
	oldMedia := utils.GetFile(oldMsg)
	mediaDiff := format.CompareMedia(oldMedia, newMedia)

	if mediaDiff.Removed == nil {
		utils.OnDataError(c, query.ID, loc)
		return fmt.Errorf("HandleEditedFiles error: no file found")
	}

	caption := ""
	if oldMsg.Caption != "" {
		caption = loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "sendMediaInGroups.caption",
			TemplateData: map[string]string{
				"Text": format.Caption(oldMsg.Caption),
			},
		})
	}

	file := utils.CreateInputMediaFromFileInfo(mediaDiff.Removed.FileID, mediaDiff.Removed.Type, caption)
	if err := utils.SendMediaInGroups(c.Bot(), c, user.ID, []telego.InputMedia{file}, query.Message.GetMessageID()); err != nil {
		utils.OnDataError(c, query.ID, loc)
		return err
	}

	return c.Bot().AnswerCallbackQuery(c, tu.CallbackQuery(query.ID))
}
