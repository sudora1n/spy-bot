package utils

import (
	"context"
	"fmt"
	"regexp"
	"ssuspy-bot/types"
	"ssuspy-common/repository/mongoRepository"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func CreateInputMediaFromFileInfo(fileID string, mediaType string, caption string) telego.InputMedia {
	return CreateInputMediaFromFileInfoByFile(tu.FileFromID(fileID), mediaType, caption, false)
}

func CreateInputMediaFromFileInfoByFile(
	file telego.InputFile,
	mediaType string,
	caption string,
	hasSpoiler bool,
) telego.InputMedia {
	switch mediaType {
	case "photo":
		return &telego.InputMediaPhoto{
			Type:       telego.MediaTypePhoto,
			Media:      file,
			Caption:    caption,
			ParseMode:  telego.ModeHTML,
			HasSpoiler: hasSpoiler,
		}
	case "video":
		return &telego.InputMediaVideo{
			Type:       telego.MediaTypeVideo,
			Media:      file,
			Caption:    caption,
			ParseMode:  telego.ModeHTML,
			HasSpoiler: hasSpoiler,
		}
	case "audio":
		return &telego.InputMediaAudio{
			Type:      telego.MediaTypeAudio,
			Media:     file,
			Caption:   caption,
			ParseMode: telego.ModeHTML,
		}
	case "animation":
		return &telego.InputMediaAnimation{
			Type:       telego.MediaTypeAnimation,
			Media:      file,
			Caption:    caption,
			ParseMode:  telego.ModeHTML,
			HasSpoiler: hasSpoiler,
		}
	default:
		return &telego.InputMediaDocument{
			Type:      telego.MediaTypeDocument,
			Media:     file,
			Caption:   caption,
			ParseMode: telego.ModeHTML,
		}
	}
}

func SortFiles(media []*types.MediaItemProcess) [][]*types.MediaItemProcess {
	groups := make(map[string][]*types.MediaItemProcess)

	for i, item := range media {
		switch item.Type {
		case "animation":
			itemType := fmt.Sprintf("%s|%d", item.Type, i)
			groups[itemType] = append(groups[itemType], item) // https://github.com/sudora1n/spy-bot/issues/29
		case "video":
			itemType := "photo"
			groups[itemType] = append(groups[itemType], item) // for grouping photo and video together
		default:
			groups[item.Type] = append(groups[item.Type], item)
		}
	}

	result := make([][]*types.MediaItemProcess, 0, len(groups))
	for _, group := range groups {
		result = append(result, group)
	}

	return result
}

func ConvertFileInfosGroupsToInputMediaGroups(mediaFiles [][]*types.MediaItemProcess) (result [][]telego.InputMedia) {
	for _, innerSlice := range mediaFiles {
		var convertedInner []telego.InputMedia
		for _, mediaItem := range innerSlice {
			if mediaItem != nil {
				inputMedia := CreateInputMediaFromFileInfo(mediaItem.FileID, mediaItem.Type, mediaItem.Caption)
				convertedInner = append(convertedInner, inputMedia)
			}
		}
		result = append(result, convertedInner)
	}
	return result
}

func BusinessMessageMatches(pattern *regexp.Regexp) th.Predicate {
	return func(_ context.Context, update telego.Update) bool {
		return update.BusinessMessage != nil && pattern.MatchString(update.BusinessMessage.Text)
	}
}

func GetBusinessRights(c *th.Context, localConnection *mongoRepository.BotUserBusinessConnection) (rights *telego.BusinessBotRights, err error) {
	if localConnection.Rights != nil {
		return localConnection.Rights, nil
	}

	connection, err := c.Bot().GetBusinessConnection(
		c,
		&telego.GetBusinessConnectionParams{
			BusinessConnectionID: localConnection.Id,
		},
	)
	if err != nil {
		return nil, err
	}
	return connection.Rights, nil
}

func OnDataError(c *th.Context, queryID string, loc *i18n.Localizer) error {
	return c.Bot().AnswerCallbackQuery(c, tu.CallbackQuery(queryID).WithText(
		loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "errors.couldNotRetrieveData",
		}),
	))
}

func OnCantReply(c *th.Context, loc *i18n.Localizer, userID int64, commandName string) error {
	_, err := c.Bot().SendMessage(
		c,
		tu.Message(
			tu.ID(userID),
			loc.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "errors.userHandlers.noCanReply",
				TemplateData: map[string]string{
					"Command": commandName,
				},
			}),
		),
	)
	return err
}

func OnFilesError(c *th.Context, userID int64, loc *i18n.Localizer, replyToMessageID int) (err error) {
	_, err = c.Bot().SendMessage(c, tu.Message(
		tu.ID(userID),
		loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "errors.errorSendingFiles",
		}),
	).WithParseMode(telego.ModeHTML).WithReplyParameters(&telego.ReplyParameters{
		MessageID:                replyToMessageID,
		ChatID:                   tu.ID(userID),
		AllowSendingWithoutReply: true,
	}))

	return err
}
