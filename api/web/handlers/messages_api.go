package handlers

import (
	"context"
	"errors"
	apiErrors "ssuspy-api/errors"
	"ssuspy-api/repository"
	"ssuspy-api/types"
	mongo_repository "ssuspy-common/repository/mongoRepository"
	"ssuspy-common/telegram/utils"
	messagesv1 "ssuspy-proto/gen/messages/v1"

	"connectrpc.com/authn"
	"connectrpc.com/connect"
	"github.com/mymmrac/telego"
)

func messageToProto(message *telego.Message) *messagesv1.Message {
	if message == nil {
		return nil
	}

	return &messagesv1.Message{
		MessageId:           int64(message.MessageID),
		From:                chatToProto(message.From),
		SenderChat:          chatToProto(message.SenderChat),
		Date:                message.Date,
		EditDate:            message.EditDate,
		Chat:                chatToProto(&message.Chat),
		ForwardOrigin:       forwardOriginToProto(message.ForwardOrigin),
		ReplyToMessage:      messageToProto(message),
		Quote:               quoteToProto(message.Quote),
		Story:               storyToProto(message.Story),
		ReplyToStory:        storyToProto(message.ReplyToStory),
		ViaBot:              chatToProto(message.ViaBot),
		HasProtectedContent: message.HasProtectedContent,
		HasMediaSpoiler:     message.HasMediaSpoiler,
		MediaGroupId:        message.MediaGroupID,
		Text:                chooseText(message),
		Entities:            entitiesToProto(message.Entities),
		Location:            locationToProto(message.Location),
		Media:               mediaToProto(message),
	}
}

func chooseText(message *telego.Message) string {
	if message.Text == "" && message.Caption == "" {
		return ""
	}

	if message.Caption != "" {
		return message.Caption
	}

	return message.Text
}

func entitiesToProto(entities []telego.MessageEntity) []*messagesv1.MessageEntity {
	entitiesResult := make([]*messagesv1.MessageEntity, len(entities))
	for i, entity := range entities {
		entitiesResult[i] = &messagesv1.MessageEntity{
			Type:     entity.Type,
			Offset:   int64(entity.Offset),
			Length:   int64(entity.Length),
			Url:      entity.URL,
			User:     chatToProto(entity.User),
			Language: entity.Language,
		}
	}

	return entitiesResult
}

func mediaToProto(message *telego.Message) (media *messagesv1.Media) {
	file := utils.GetFile(message)

	if file == nil {
		return nil
	}

	media = &messagesv1.Media{
		Type:     messagesv1.MediaType_MEDIA_TYPE_UNSPECIFIED,
		FileId:   file.FileID,
		FileSize: file.FileSize,
	}

	switch file.Type {
	case "photo":
		media.Type = messagesv1.MediaType_MEDIA_TYPE_PHOTO
	case "video":
		media.Type = messagesv1.MediaType_MEDIA_TYPE_VIDEO
	case "animation":
		media.Type = messagesv1.MediaType_MEDIA_TYPE_ANIMATION
	case "audio":
		media.Type = messagesv1.MediaType_MEDIA_TYPE_AUDIO
	case "voice":
		media.Type = messagesv1.MediaType_MEDIA_TYPE_VOICE
	case "document":
		media.Type = messagesv1.MediaType_MEDIA_TYPE_DOCUMENT
	case "video_note":
		media.Type = messagesv1.MediaType_MEDIA_TYPE_VIDEO_NOTE
	case "sticker":
		media.Type = messagesv1.MediaType_MEDIA_TYPE_STICKER
	}

	return media
}

func locationToProto(location *telego.Location) *messagesv1.Location {
	if location == nil {
		return nil
	}

	return &messagesv1.Location{
		Latitude:  location.Latitude,
		Longitude: location.Longitude,
	}
}

func quoteToProto(quote *telego.TextQuote) *messagesv1.Quote {
	if quote == nil {
		return nil
	}

	return &messagesv1.Quote{
		Text:     quote.Text,
		Entities: entitiesToProto(quote.Entities),
		Position: int64(quote.Position),
		IsManual: quote.IsManual,
	}
}

func storyToProto(story *telego.Story) *messagesv1.Story {
	if story == nil {
		return nil
	}

	return &messagesv1.Story{
		Chat: chatToProto(&story.Chat),
		Id:   int64(story.ID),
	}
}

func forwardOriginToProto(forwardOrigin telego.MessageOrigin) *messagesv1.ForwardOrigin {
	var result messagesv1.ForwardOrigin

	switch v := forwardOrigin.(type) {
	case *telego.MessageOriginUser:
		result.ForwardOrigin = &messagesv1.ForwardOrigin_User{
			User: &messagesv1.ForwardOriginUser{
				Date:       v.Date,
				SenderUser: chatToProto(&v.SenderUser),
			},
		}
	case *telego.MessageOriginHiddenUser:
		result.ForwardOrigin = &messagesv1.ForwardOrigin_HiddenUser{
			HiddenUser: &messagesv1.ForwardOriginHiddenUser{
				Date:           v.Date,
				SenderUsername: v.SenderUserName,
			},
		}
	case *telego.MessageOriginChat:
		result.ForwardOrigin = &messagesv1.ForwardOrigin_Chat{
			Chat: &messagesv1.ForwardOriginChat{
				Date:            v.Date,
				SenderChat:      chatToProto(&v.SenderChat),
				AuthorSignature: v.AuthorSignature,
			},
		}
	case *telego.MessageOriginChannel:
		result.ForwardOrigin = &messagesv1.ForwardOrigin_Channel{
			Channel: &messagesv1.ForwardOriginChannel{
				Date:            v.Date,
				Chat:            chatToProto(&v.Chat),
				MessageId:       int64(v.MessageID),
				AuthorSignature: v.AuthorSignature,
			},
		}
	default:
		return nil
	}

	return &result
}

func chatToProto[T *telego.Chat | *telego.User](entity T) *messagesv1.Chat {
	switch v := any(entity).(type) {
	case telego.Chat:
		return &messagesv1.Chat{
			Id:        v.ID,
			Type:      v.Type,
			Title:     v.Title,
			FirstName: v.FirstName,
			LastName:  v.LastName,
			Username:  v.Username,
		}
	case telego.User:
		return &messagesv1.Chat{
			Id:        v.ID,
			Type:      telego.ChatTypePrivate,
			FirstName: v.FirstName,
			LastName:  v.LastName,
			Username:  v.Username,
		}
	default:
		return nil
	}
}

var (
	ErrMessagesNotFound = errors.New("messages not found")

	ConnectErrMessagesNotFound = connect.NewError(connect.CodeNotFound, ErrMessagesNotFound)
)

type MessagesServer struct {
	repo *repository.Repository
}

func NewMessagesServer(repo *repository.Repository) *MessagesServer {
	return &MessagesServer{
		repo: repo,
	}
}

func (s *MessagesServer) GetMessages(
	ctx context.Context,
	req *connect.Request[messagesv1.GetMessagesRequest],
) (*connect.Response[messagesv1.GetMessagesResponse], error) {
	user := authn.GetInfo(ctx).(*types.User)

	messageIds := make([]int, len(req.Msg.MessageIds))
	for i, msg := range req.Msg.MessageIds {
		messageIds[i] = int(msg)
	}

	msgRes, err := s.repo.Mongo.GetMessages(ctx, &mongo_repository.GetMessagesOptions{
		UserID:     user.Id,
		PeerID:     req.Msg.ChatId,
		MessageIDs: messageIds,
		WithEdits:  req.Msg.WithEdits,
		Offset:     int(req.Msg.Offset),
		Limit:      int(req.Msg.Limit),
	})
	if err != nil {
		return nil, apiErrors.CONNECT_ERROR_INTERNAL
	}
	if len(msgRes.Messages) == 0 {
		return nil, ConnectErrMessagesNotFound
	}

	messagesResult := make([]*messagesv1.Message, len(msgRes.Messages))
	for i, message := range msgRes.Messages {
		messagesResult[i] = messageToProto(message)
	}

	res := connect.NewResponse(&messagesv1.GetMessagesResponse{
		Messages:    messagesResult,
		HasForward:  msgRes.Pagination.Forward,
		HasBackward: msgRes.Pagination.Forward,
	})
	res.Header().Set("GetMessages-Version", "v1")
	return res, nil
}
