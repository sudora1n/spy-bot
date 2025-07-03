package format

import (
	"fmt"
	"html"
	"ssuspy-bot/consts"
	"ssuspy-bot/telegram/utils"
	"ssuspy-bot/types"
	"strconv"
	"strings"
	"unicode/utf8"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func CompareMedia(oldMedia, newMedia *types.MediaItem) (diff types.MediaDiff) {
	if oldMedia == nil && newMedia == nil {
		return diff
	}
	if oldMedia == nil {
		if newMedia.FileID != "" {
			diff.Added = newMedia
		}
		return diff
	}
	if newMedia == nil {
		if oldMedia.FileID != "" {
			diff.Removed = oldMedia
		}
		return diff
	}

	if oldMedia.FileID == newMedia.FileID && oldMedia.Type == newMedia.Type {
		return diff
	}

	if oldMedia.FileID != "" {
		diff.Removed = oldMedia
	}
	if newMedia.FileID != "" {
		diff.Added = newMedia
	}

	return diff
}

func SummarizeDeletedMessage(message *telego.Message, loc *i18n.Localizer, truncate bool) string {
	var summary []string

	if message.ForwardOrigin != nil {
		forwardInfo := getForwardInfo(message, loc)
		if forwardInfo != "" {
			summary = append(
				summary,
				loc.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "business.deleted.format.forwardInfo.isForwardedWithInfo",
					TemplateData: map[string]string{
						"Info": forwardInfo,
					},
				}),
			)
		} else {
			summary = append(
				summary,
				loc.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "business.deleted.format.forwardInfo.isForwarded",
				}),
			)
		}
	}

	var text string
	switch {
	case message.Text != "":
		text = message.Text
	case message.Caption != "":
		text = message.Caption
	}

	if text != "" {
		summary = append(
			summary,
			loc.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "business.deleted.format.text",
				TemplateData: map[string]string{
					"Text": formatText(text, truncate),
				},
			}),
		)
	}

	var media []string
	switch {
	case message.Photo != nil:
		media = append(media, "photo")
	case message.Video != nil:
		media = append(media, "video")
	case message.Animation != nil:
		media = append(media, "animation")
	case message.Audio != nil:
		media = append(media, "audio")
	case message.Voice != nil:
		media = append(media, "voice")
	case message.Document != nil:
		media = append(media, "document")
	case message.Sticker != nil:
		media = append(media, "sticker")
	case message.VideoNote != nil:
		media = append(media, "video_note")
	}

	for _, value := range media {
		mediaText := loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: fmt.Sprintf("mediaTypes.%s", value),
		})

		summary = append(
			summary,
			loc.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "business.deleted.format.media",
				TemplateData: map[string]string{
					"Media": mediaText,
				},
			}),
		)
	}

	if message.Location != nil {
		lat := strconv.FormatFloat(message.Location.Latitude, 'f', -1, 64)
		lon := strconv.FormatFloat(message.Location.Longitude, 'f', -1, 64)
		summary = append(
			summary,
			loc.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "business.deleted.format.location",
				TemplateData: map[string]any{
					"Latitude":  lat,
					"Longitude": lon,
				},
			}),
		)
	}

	if len(summary) == 0 {
		return loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "business.deleted.format.empty",
		})
	}

	return strings.Join(summary, "\n")
}

func SummarizeDeletedMessages(messages []*telego.Message, name string, loc *i18n.Localizer, truncate bool, offset int, messagesLen int) string {
	if messagesLen == 1 {
		return loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "business.deleted.format.message",
			TemplateData: map[string]string{
				"Result":           SummarizeDeletedMessage(messages[0], loc, truncate),
				"ResolvedChatName": name,
			},
			PluralCount: messagesLen,
		})
	}

	var result strings.Builder
	for i, message := range messages {
		summarize := SummarizeDeletedMessage(message, loc, truncate)
		result.WriteString(loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "business.deleted.format.messageItem",
			TemplateData: map[string]any{
				"Count":   i + 1 + offset,
				"Message": summarize,
			},
		}))
	}

	return loc.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "business.deleted.format.message",
		TemplateData: map[string]any{
			"Count":            messagesLen,
			"Result":           result.String(),
			"ResolvedChatName": name,
		},
		PluralCount: messagesLen,
	})
}

func EditedDiff(oldMsg *telego.Message, newMsg *telego.Message, loc *i18n.Localizer, truncate bool) ([]string, types.MediaDiff) {
	var changes []string

	oldHasText := oldMsg.Text != ""
	newHasText := newMsg.Text != ""
	textEqual := newMsg.Text == oldMsg.Text

	if oldHasText && newHasText && !textEqual {
		changes = append(
			changes,
			loc.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "business.edited.text.changed",
				TemplateData: map[string]string{
					"Old": formatText(oldMsg.Text, truncate),
					"New": formatText(newMsg.Text, truncate),
				},
			}),
		)
	} else if !oldHasText && newHasText {
		changes = append(
			changes,
			loc.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "business.edited.text.added",
				TemplateData: map[string]string{
					"New": formatText(newMsg.Text, truncate),
				},
			}),
		)
	} else if oldHasText && !newHasText && newMsg.Caption != oldMsg.Text {
		changes = append(
			changes,
			loc.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "business.edited.text.removed",
				TemplateData: map[string]string{
					"New": formatText(oldMsg.Text, truncate),
				},
			}),
		)
	}

	oldHasCaption := oldMsg.Caption != ""
	newHasCaption := newMsg.Caption != ""
	captionEqual := newMsg.Caption == oldMsg.Caption

	if oldHasCaption && newHasCaption && !captionEqual {
		changes = append(
			changes,
			loc.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "business.edited.text.changed",
				TemplateData: map[string]string{
					"Old": formatText(oldMsg.Caption, truncate),
					"New": formatText(newMsg.Caption, truncate),
				},
			}),
		)
	} else if !oldHasCaption && newHasCaption && newMsg.Caption != oldMsg.Text {
		changes = append(
			changes,
			loc.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "business.edited.text.added",
				TemplateData: map[string]string{
					"New": formatText(newMsg.Caption, truncate),
				},
			}),
		)
	} else if oldHasCaption && !newHasCaption {
		changes = append(
			changes,
			loc.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "business.edited.text.removed",
				TemplateData: map[string]string{
					"New": formatText(oldMsg.Caption, truncate),
				},
			}),
		)
	}

	oldMedia := utils.GetFile(oldMsg)
	newMedia := utils.GetFile(newMsg)
	mediaDiff := CompareMedia(oldMedia, newMedia)

	if mediaDiff.Added != nil || mediaDiff.Removed != nil {
		var (
			msgID     string
			mediaType string
		)

		switch {
		case (mediaDiff.Added != nil && mediaDiff.Removed != nil) && (mediaDiff.Added.Type == mediaDiff.Removed.Type):
			msgID = "business.edited.media.updated"
			mediaType = mediaDiff.Added.Type
		case mediaDiff.Added != nil:
			msgID = "business.edited.media.added"
			mediaType = mediaDiff.Added.Type
		case mediaDiff.Removed != nil:
			msgID = "business.edited.media.removed"
			mediaType = mediaDiff.Removed.Type
		}

		locMediaType := loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: fmt.Sprintf("mediaTypes.%s", mediaType),
		})

		changes = append(changes, loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: msgID,
			TemplateData: map[string]string{
				"MediaType": locMediaType,
			},
		}))
	}

	return changes, mediaDiff
}

func TruncateText(text string, maxLength int, replaceN bool) (result string) {
	return CustomTruncateText(text, maxLength, "...", replaceN)
}

func CustomTruncateText(text string, maxLength int, endString string, replaceN bool) (result string) {
	if replaceN {
		text = strings.ReplaceAll(text, "\n", " ")
	}

	if maxLength <= 0 {
		return ""
	}

	if utf8.RuneCountInString(text) <= maxLength {
		return text
	}

	runes := []rune(text)

	endStringLen := utf8.RuneCountInString(endString)
	result = string(runes[:maxLength-endStringLen])
	return result + endString
}

func getForwardInfo(msg *telego.Message, loc *i18n.Localizer) string {
	if msg.ForwardOrigin == nil {
		return ""
	}

	switch origin := msg.ForwardOrigin.(type) {
	case *telego.MessageOriginUser:
		u := origin.SenderUser
		name := Name(u.FirstName, u.LastName)
		if u.Username != "" {
			return loc.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "business.deleted.format.forwardInfo.user",
				TemplateData: map[string]string{
					"Name":     name,
					"Username": u.Username,
				},
			})
		}
		return loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "business.deleted.format.forwardInfo.hiddenUser",
			TemplateData: map[string]string{
				"Name": name,
			},
		})

	case *telego.MessageOriginHiddenUser:
		return loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "business.deleted.format.forwardInfo.hiddenUser",
			TemplateData: map[string]string{
				"Name": origin.SenderUserName,
			},
		})

	case *telego.MessageOriginChat:
		ch := origin.SenderChat
		return loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "business.deleted.format.forwardInfo.chat",
			TemplateData: map[string]any{
				"Title": ch.Title,
				"ID":    ch.ID,
			},
		})

	case *telego.MessageOriginChannel:
		if origin.Chat.Username != "" {
			return loc.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "business.deleted.format.forwardInfo.channel",
				TemplateData: map[string]string{
					"Title":    origin.Chat.Title,
					"Username": origin.Chat.Username,
				},
			})
		}
		return loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "business.deleted.format.forwardInfo.hiddenChannel",
			TemplateData: map[string]string{
				"Title": origin.Chat.Title,
			},
		})

	default:
		return ""
	}
}

func GetMDInputFile(text string, fileName string) telego.InputFile {
	text = strings.ReplaceAll(text, "\n", "<br>")
	newText, err := htmltomarkdown.ConvertString(text)
	if err == nil {
		fileName += ".md"
	} else {
		fileName += ".txt"
		newText = text
	}

	reader := strings.NewReader(newText)
	return tu.FileFromReader(reader, fileName)
}

func FilterMessagesByDate(msgs []*telego.Message) []*telego.Message {
	uniq := make(map[int]*telego.Message)

	for _, m := range msgs {
		id := m.MessageID
		if exist, ok := uniq[id]; !ok {
			uniq[id] = m
		} else {
			tNew := chooseTime(m.EditDate, m.Date)
			tOld := chooseTime(exist.EditDate, exist.Date)

			if tNew > tOld {
				uniq[id] = m
			}
		}
	}

	result := make([]*telego.Message, 0, len(uniq))
	for _, m := range uniq {
		result = append(result, m)
	}

	return result
}

func chooseTime(editDate, date int64) int64 {
	if editDate > 0 {
		return editDate
	}
	return date
}

func Name(name string, lastName string) string {
	if lastName != "" {
		name += fmt.Sprintf(" %s", lastName)
	}
	name = html.EscapeString(
		TruncateText(name, consts.MAX_NAME_LEN, true),
	)

	return name
}

func formatText(text string, truncate bool) string {
	if truncate {
		text = TruncateText(text, consts.MAX_MESSAGE_TEXT_LEN, true)
	}
	return html.EscapeString(
		text,
	)
}

func Caption(text string) string {
	return html.EscapeString(
		TruncateText(text, consts.MAX_MEDIA_CAPTION_LEN, false),
	)
}
