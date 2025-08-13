package mongoRepository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mymmrac/telego"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrMessageIsNil = errors.New("message is nil")
)

type Message struct {
	ID       primitive.ObjectID   `bson:"_id"`
	ChatIDs  []primitive.ObjectID `bson:"chat_ids"`
	Messages []bson.Raw           `bson:"messages"`
}

type GetMessageOptions struct {
	UserID    int64
	PeerID    int64
	MessageID int
}

type GetMessagesOptions struct {
	UserID     int64
	PeerID     int64
	MessageIDs []int
	WithEdits  bool
	Offset     int
	Limit      int
}

type GetMessagesResponse struct {
	Messages   []*telego.Message
	Pagination *PaginationAnswer
}

type PaginationAnswer struct {
	Forward  bool
	Backward bool
}

func (r *MongoRepository) SaveMessage(ctx context.Context, message *telego.Message, userID int64) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if message == nil {
		return ErrMessageIsNil
	}

	msg := *message

	session, err := r.client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	callback := func(sessCtx mongo.SessionContext) (any, error) {
		chat, err := r.syncAndGetChat(sessCtx, userID, &msg)
		if err != nil {
			return nil, fmt.Errorf("failed to upsert chat: %w", err)
		}

		err = r.syncPeer(sessCtx, &Peer{
			ID:        msg.Chat.ID,
			FirstName: msg.Chat.FirstName,
			LastName:  msg.Chat.LastName,
			Username:  msg.Chat.Username,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to upsert peer: %w", err)
		}

		msg.BusinessConnectionID, msg.Chat = "", telego.Chat{}
		msgBytes, err := r.customRegistry.SaveMessage(&msg)
		if err != nil {
			return nil, err
		}
		msgRaw := bson.Raw(msgBytes)

		_, err = r.telegramMessages.UpdateOne(
			sessCtx,
			bson.M{
				"messages.message_id": msg.MessageID,
				"chat_ids":            chat.ID,
			},
			bson.D{
				{Key: "$addToSet", Value: bson.D{
					{Key: "chat_ids", Value: chat.ID},
					{Key: "messages", Value: msgRaw},
				}},
			},

			options.Update().SetUpsert(true),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to upsert message: %w", err)
		}

		return nil, nil
	}

	_, err = session.WithTransaction(ctx, callback)
	return err
}

func (r *MongoRepository) GetMessage(
	ctx context.Context,
	options *GetMessageOptions,
) (*telego.Message, error) {
	res, err := r.GetMessages(
		ctx,
		&GetMessagesOptions{
			UserID:     options.UserID,
			PeerID:     options.PeerID,
			MessageIDs: []int{options.MessageID},
		},
	)
	if err != nil {
		return nil, err
	}
	if len(res.Messages) == 0 {
		return nil, errors.New("no data found for the specified messageID")
	}

	return res.Messages[0], nil
}

func (r *MongoRepository) GetMessages(ctx context.Context, options *GetMessagesOptions) (*GetMessagesResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	chatWithPeer, err := r.getChatWithPeer(ctx, options.UserID, options.PeerID)
	if err != nil {
		return nil, fmt.Errorf("failed to find peer and peer: %w", err)
	}

	pipeline := mongo.Pipeline{}

	matchConditions := bson.M{
		"chat_ids": chatWithPeer.Chat.ID,
	}

	if len(options.MessageIDs) > 0 {
		matchConditions["messages.message_id"] = bson.M{"$in": options.MessageIDs}
	}

	pipeline = append(pipeline, bson.D{{Key: "$match", Value: matchConditions}})

	pipeline = append(pipeline, bson.D{{Key: "$unwind", Value: "$messages"}})

	if options.WithEdits {
		pipeline = append(pipeline, bson.D{{Key: "$sort", Value: bson.D{
			{Key: "messages.message_id", Value: 1},
			{Key: "messages.edit_date", Value: -1},
			{Key: "messages.date", Value: -1},
		}}})
	} else {
		pipeline = append(pipeline, bson.D{{Key: "$sort", Value: bson.D{
			{Key: "messages.message_id", Value: 1},
			{Key: "messages.edit_date", Value: -1},
			{Key: "messages.date", Value: -1},
		}}})

		pipeline = append(pipeline, bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$messages.message_id"},
			{Key: "latest_message", Value: bson.D{{Key: "$first", Value: "$messages"}}},
		}}})

		pipeline = append(pipeline, bson.D{{Key: "$replaceRoot", Value: bson.D{
			{Key: "newRoot", Value: bson.D{
				{Key: "messages", Value: "$latest_message"},
			}},
		}}})

		pipeline = append(pipeline, bson.D{{Key: "$sort", Value: bson.D{{Key: "messages.message_id", Value: 1}}}})
	}

	if options.Offset > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$skip", Value: options.Offset}})
	}

	if options.Limit > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$limit", Value: options.Limit + 1}})
	}

	cursor, err := r.telegramMessages.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate messages: %w", err)
	}
	defer cursor.Close(ctx)

	var results []struct {
		MessageID int64                `bson:"messages.message_id"`
		Messages  bson.Raw             `bson:"messages"`
		ChatIDs   []primitive.ObjectID `bson:"chat_ids"`
	}

	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	var messages []*telego.Message
	for _, result := range results {
		var msg telego.Message
		if err := r.customRegistry.LoadMessage(result.Messages, &msg); err != nil {
			log.Warn().Err(err).Msg("fail decode message")
			continue
		}

		msg.Chat = telego.Chat{
			ID:        chatWithPeer.Peer.ID,
			Type:      telego.ChatTypePrivate,
			Username:  chatWithPeer.Peer.Username,
			FirstName: chatWithPeer.Peer.FirstName,
			LastName:  chatWithPeer.Peer.LastName,
		}

		messages = append(messages, &msg)
	}

	pagination := PaginationAnswer{
		Backward: options.Offset > 0,
	}
	if options.Limit > 0 && len(messages) > options.Limit {
		pagination.Forward = true
		messages = messages[:len(messages)-1]
	}

	return &GetMessagesResponse{
		Messages:   messages,
		Pagination: &pagination,
	}, nil
}
