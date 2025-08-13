package sendMedia

import (
	"context"
	"errors"
	"fmt"
	"ssuspy-bot/consts"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/rs/zerolog/log"
)

var (
	ErrMediaNoMatch = errors.New("media doesnt match any filters")
)

func SendOneMedia(
	ctx context.Context,
	bot *telego.Bot,
	userID int64,
	media telego.InputMedia,
	replyMessageID int,
) (err error) {
	replyParams := &telego.ReplyParameters{
		ChatID:                   tu.ID(userID),
		MessageID:                replyMessageID,
		AllowSendingWithoutReply: true,
	}

	switch media := media.(type) {
	case *telego.InputMediaPhoto:
		_, err = bot.SendPhoto(ctx, &telego.SendPhotoParams{
			ChatID:          tu.ID(userID),
			Photo:           media.Media,
			Caption:         media.Caption,
			ParseMode:       media.ParseMode,
			ReplyParameters: replyParams,
			HasSpoiler:      media.HasSpoiler,
		})
	case *telego.InputMediaVideo:
		_, err = bot.SendVideo(ctx, &telego.SendVideoParams{
			ChatID:          tu.ID(userID),
			Video:           media.Media,
			Caption:         media.Caption,
			ParseMode:       media.ParseMode,
			ReplyParameters: replyParams,
			HasSpoiler:      media.HasSpoiler,
		})
	case *telego.InputMediaDocument:
		_, err = bot.SendDocument(ctx, &telego.SendDocumentParams{
			ChatID:          tu.ID(userID),
			Document:        media.Media,
			Caption:         media.Caption,
			ParseMode:       media.ParseMode,
			ReplyParameters: replyParams,
		})
	case *telego.InputMediaAudio:
		_, err = bot.SendAudio(ctx, &telego.SendAudioParams{
			ChatID:          tu.ID(userID),
			Audio:           media.Media,
			Caption:         media.Caption,
			ParseMode:       media.ParseMode,
			ReplyParameters: replyParams,
		})
	case *telego.InputMediaAnimation:
		_, err = bot.SendAnimation(ctx, &telego.SendAnimationParams{
			ChatID:          tu.ID(userID),
			Animation:       media.Media,
			Caption:         media.Caption,
			ParseMode:       media.ParseMode,
			ReplyParameters: replyParams,
			HasSpoiler:      media.HasSpoiler,
		})
	default:
		err = ErrMediaNoMatch
	}

	return err
}

func SendMediaInGroups(
	bot *telego.Bot,
	ctx context.Context,
	userID int64,
	mediaItems []telego.InputMedia,
	replyMessageID int,
) error {
	for i := 0; i < len(mediaItems); i += consts.MAX_MEDIA_GROUP_SIZE {
		end := min(i+consts.MAX_MEDIA_GROUP_SIZE, len(mediaItems))
		batch := mediaItems[i:end]
		if len(batch) == 0 {
			continue
		}

		if len(batch) == 1 {
			item := batch[0]

			err := SendOneMedia(ctx, bot, userID, item, replyMessageID)
			if err != nil {
				log.Warn().Err(err).Int64("userID", userID).Msg("error sending single media item")
				return fmt.Errorf("failed to send single media item %v: %w", item, err)
			}
		} else {
			replyParams := &telego.ReplyParameters{
				ChatID:                   tu.ID(userID),
				MessageID:                replyMessageID,
				AllowSendingWithoutReply: true,
			}
			_, err := bot.SendMediaGroup(ctx, tu.MediaGroup(tu.ID(userID), batch...).
				WithReplyParameters(replyParams))
			if err != nil {
				log.Warn().Err(err).Int64("userID", userID).Msg("error sending media group")
				return fmt.Errorf("failed to send media group: %w", err)
			}
		}

		time.Sleep(500 * time.Millisecond)
	}
	return nil
}
