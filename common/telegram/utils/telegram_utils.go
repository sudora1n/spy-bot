package utils

import (
	"ssuspy-bot/types"

	"github.com/mymmrac/telego"
)

func GetFile(message *telego.Message) (media *types.MediaItem) {
	if message == nil {
		return nil
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
