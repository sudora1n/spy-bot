package api

import (
	"context"
	"ssuspy-api/errors"
	"ssuspy-api/repository"
	messagesv1 "ssuspy-proto/gen/messages/v1"

	"connectrpc.com/connect"
	"go.mongodb.org/mongo-driver/bson"
)

type MessagesServer struct {
	mongo repository.MongoRepository
}

func (s *MessagesServer) CreateMessage(
	ctx context.Context,
	req *connect.Request[messagesv1.CreateMessageRequest],
) (*connect.Response[messagesv1.CreateMessageResponse], error) {
	err := s.mongo.SaveMessage(ctx, req.Msg.Payload)
	if err != nil {
		return nil, errors.CONNECT_ERROR_INTERNAL
	}

	// name := format.Name(
	// 	message.Chat.FirstName,
	// 	message.Chat.LastName,
	// )

	// err = h.service.UpdateChatName(
	// 	c,
	// 	message.Chat.ID,
	// 	name,
	// )
	// if err != nil {
	// 	log.Warn().Err(err).Msg("failed save/update chat name")
	// }

	res := connect.NewResponse(&messagesv1.CreateMessageResponse{})
	res.Header().Set("CreateMessage-Version", "v1")
	return res, nil
}

func (s *MessagesServer) GetMessage(
	ctx context.Context,
	req *connect.Request[messagesv1.GetMessageRequest],
) (*connect.Response[messagesv1.GetMessageResponse], error) {
	msg, err := s.mongo.GetMessage(ctx, &repository.GetMessageOptions{
		ChatID:        req.Msg.ChatId,
		MessageID:     int(req.Msg.MessageId),
		ConnectionIDs: req.Msg.ConnectionIds,
	})
	if err != nil {
		return nil, errors.CONNECT_ERROR_INTERNAL
	}

	payload, err := bson.Marshal(msg)
	if err != nil {
		return nil, errors.CONNECT_ERROR_INTERNAL
	}

	res := connect.NewResponse(&messagesv1.GetMessageResponse{
		Payload: payload,
	})
	res.Header().Set("GetMessage-Version", "v1")
	return res, nil
}

func (s *MessagesServer) GetMessages(
	ctx context.Context,
	req *connect.Request[messagesv1.GetMessagesRequest],
) (*connect.Response[messagesv1.GetMessagesResponse], error) {
	messageIds := make([]int, len(req.Msg.MessageIds))

	for i, msg := range req.Msg.MessageIds {
		messageIds[i] = int(msg)
	}

	msgs, pagination, err := s.mongo.GetMessages(ctx, &repository.GetMessagesOptions{
		ChatID:        req.Msg.ChatId,
		MessageIDs:    messageIds,
		ConnectionIDs: req.Msg.ConnectionIds,
		WithEdits:     req.Msg.WithEdits,
		Offset:        int(req.Msg.Offset),
		Limit:         int(req.Msg.Limit),
	})
	if err != nil {
		return nil, errors.CONNECT_ERROR_INTERNAL
	}

	payload := make([][]byte, len(msgs))
	for _, msg := range msgs {
		data, err := bson.Marshal(msg)
		if err != nil {
			return nil, errors.CONNECT_ERROR_INTERNAL
		}

		payload = append(payload, data)
	}

	res := connect.NewResponse(&messagesv1.GetMessagesResponse{
		Payload:     payload,
		HasForward:  pagination.Forward,
		HasBackward: pagination.Backward,
	})
	res.Header().Set("GetMessages-Version", "v1")
	return res, nil
}
