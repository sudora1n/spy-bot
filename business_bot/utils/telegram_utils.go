package utils

import (
	"context"
	"fmt"
	"regexp"
	"ssuspy-bot/consts"
	"ssuspy-bot/repository"
	"ssuspy-bot/types"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/rs/zerolog/log"
)

func SendMediaInGroups(bot *telego.Bot, ctx context.Context, userID int64, mediaItems []telego.InputMedia, ReplyMessageID int) error {
	replyParams := &telego.ReplyParameters{
		ChatID:                   tu.ID(userID),
		MessageID:                ReplyMessageID,
		AllowSendingWithoutReply: true,
	}

	for i := 0; i < len(mediaItems); i += consts.MAX_MEDIA_GROUP_SIZE {
		end := min(i+consts.MAX_MEDIA_GROUP_SIZE, len(mediaItems))
		batch := mediaItems[i:end]
		if len(batch) == 0 {
			continue
		}

		if len(batch) == 1 {
			item := batch[0]
			var err error
			switch media := item.(type) {
			case *telego.InputMediaPhoto:
				_, err = bot.SendPhoto(ctx,
					tu.Photo(tu.ID(userID), media.Media).
						WithCaption(media.Caption).
						WithParseMode(media.ParseMode).
						WithReplyParameters(replyParams),
				)
			case *telego.InputMediaVideo:
				_, err = bot.SendVideo(ctx,
					tu.Video(tu.ID(userID), media.Media).
						WithCaption(media.Caption).
						WithParseMode(media.ParseMode).
						WithReplyParameters(replyParams),
				)
			case *telego.InputMediaDocument:
				_, err = bot.SendDocument(ctx,
					tu.Document(tu.ID(userID), media.Media).
						WithCaption(media.Caption).
						WithParseMode(media.ParseMode).
						WithReplyParameters(replyParams),
				)
			case *telego.InputMediaAudio:
				_, err = bot.SendAudio(ctx,
					tu.Audio(tu.ID(userID), media.Media).
						WithCaption(media.Caption).
						WithParseMode(media.ParseMode).
						WithReplyParameters(replyParams),
				)
			case *telego.InputMediaAnimation:
				_, err = bot.SendAnimation(ctx,
					tu.Animation(tu.ID(userID), media.Media).
						WithCaption(media.Caption).
						WithParseMode(media.ParseMode).
						WithReplyParameters(replyParams),
				)
			}

			if err != nil {
				log.Warn().Err(err).Int64("userID", userID).Msg("Error sending single media item")
				return fmt.Errorf("failed to send single media item %v: %w", item, err)
			}
		} else {
			_, err := bot.SendMediaGroup(ctx, tu.MediaGroup(tu.ID(userID), batch...).
				WithReplyParameters(replyParams))
			if err != nil {
				log.Warn().Err(err).Int64("userID", userID).Msg("Error sending media group")
				return fmt.Errorf("failed to send media group: %w", err)
			}
		}

		time.Sleep(500 * time.Millisecond)
	}
	return nil
}

func CreateInputMediaFromFileInfo(fileID string, mediaType string, caption string) telego.InputMedia {
	return CreateInputMediaFromFileInfoByFile(tu.FileFromID(fileID), mediaType, caption)
}

func CreateInputMediaFromFileInfoByFile(file telego.InputFile, mediaType string, caption string) telego.InputMedia {
	switch mediaType {
	case "photo":
		return tu.MediaPhoto(file).WithCaption(caption).WithParseMode(telego.ModeHTML)
	case "video":
		return tu.MediaVideo(file).WithCaption(caption).WithParseMode(telego.ModeHTML)
	case "audio":
		return tu.MediaAudio(file).WithCaption(caption).WithParseMode(telego.ModeHTML)
	case "animation":
		return tu.MediaAnimation(file).WithCaption(caption).WithParseMode(telego.ModeHTML)
	default:
		return tu.MediaDocument(file).WithCaption(caption).WithParseMode(telego.ModeHTML)
	}
}

func GetFile(message *telego.Message) (media *types.MediaItem) {
	if message == nil {
		return media
	}

	switch {
	case len(message.Photo) > 0:
		actualFile := message.Photo[len(message.Photo)-1]
		media = &types.MediaItem{
			Type:     "photo",
			FileID:   actualFile.FileID,
			FileSize: int64(actualFile.FileSize),
		}
		break
	case message.Video != nil:
		media = &types.MediaItem{
			Type:     "video",
			FileID:   message.Video.FileID,
			FileSize: message.Video.FileSize,
		}
		break
	case message.Animation != nil:
		media = &types.MediaItem{
			Type:     "animation",
			FileID:   message.Animation.FileID,
			FileSize: message.Animation.FileSize,
		}
		break
	case message.Audio != nil:
		media = &types.MediaItem{
			Type:     "audio",
			FileID:   message.Audio.FileID,
			FileSize: message.Audio.FileSize,
		}
		break
	case message.Voice != nil:
		media = &types.MediaItem{
			Type:     "voice",
			FileID:   message.Voice.FileID,
			FileSize: message.Voice.FileSize,
		}
		break
	case message.Document != nil:
		media = &types.MediaItem{
			Type:     "document",
			FileID:   message.Document.FileID,
			FileSize: message.Document.FileSize,
		}
		break
	case message.Sticker != nil:
		media = &types.MediaItem{
			Type:     "sticker",
			FileID:   message.Sticker.FileID,
			FileSize: int64(message.Sticker.FileSize),
		}
		break
	case message.VideoNote != nil:
		media = &types.MediaItem{
			Type:     "video_note",
			FileID:   message.VideoNote.FileID,
			FileSize: int64(message.VideoNote.FileSize),
		}
		break
	}

	return media
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

func GetBusinessRights(c *th.Context, localConnection *repository.BusinessConnection) (rights *telego.BusinessBotRights, err error) {
	if localConnection.Rights == nil {
		connection, err := c.Bot().GetBusinessConnection(
			c,
			&telego.GetBusinessConnectionParams{
				BusinessConnectionID: localConnection.ID,
			},
		)
		if err != nil {
			return nil, err
		}
		return connection.Rights, nil
	}
	return localConnection.Rights, nil
}

func OnDataError(c *th.Context, queryID string, loc *i18n.Localizer) {
	c.Bot().AnswerCallbackQuery(c, tu.CallbackQuery(queryID).WithText(
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

func OnFilesError(c *th.Context, userID int64, loc *i18n.Localizer, replyToMessageID int) {
	c.Bot().SendMessage(c, tu.Message(
		tu.ID(userID),
		loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "errors.errorSendingFiles",
		}),
	).WithParseMode(telego.ModeHTML).WithReplyParameters(&telego.ReplyParameters{
		MessageID:                replyToMessageID,
		ChatID:                   tu.ID(userID),
		AllowSendingWithoutReply: true,
	}))
}
