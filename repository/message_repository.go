package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/mymmrac/telego"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type internalMessage struct {
	InternalID           int64  `bson:"_id"`
	ChatID               int64  `bson:"chat_id"`
	MessageID            int    `bson:"message_id"`
	BusinessConnectionID string `bson:"business_connection"`
	Json                 string `bson:"json"`

	EditDate int64 `bson:"edit_date,omitempty"`
	Date     int64 `bson:"date"`

	MediaGroupID string `bson:"media_group_id,omitempty"`
}

type GetMessageOptions struct {
	ChatID        int64
	MessageID     int
	ConnectionIDs []string
}

type GetMessagesOptions struct {
	ChatID        int64
	MessageIDs    []int
	ConnectionIDs []string
	WithEdits     bool
	Offset        int
	Limit         int
}

type PaginationAnswer struct {
	Forward  bool
	Backward bool
}

func (r *MongoRepository) SaveMessage(ctx context.Context, message telego.Message) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	jsonBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message to JSON: %w", err)
	}

	_id, err := r.GetNextSequence(ctx, r.telegramMessages.Name())
	if err != nil {
		return fmt.Errorf("failed get next seq: %w", err)
	}

	doc := internalMessage{
		InternalID:           _id.Value,
		ChatID:               message.Chat.ID,
		MessageID:            message.MessageID,
		BusinessConnectionID: message.BusinessConnectionID,
		Json:                 string(jsonBytes),
		EditDate:             message.EditDate,
		Date:                 message.Date,
		MediaGroupID:         message.MediaGroupID,
	}

	_, err = r.telegramMessages.InsertOne(ctx, doc)
	if err != nil {
		return fmt.Errorf("failed to save/update message in MongoDB: %w", err)
	}
	return nil
}

func (r *MongoRepository) GetMessage(
	ctx context.Context,
	options *GetMessageOptions,
) (*telego.Message, error) {
	messages, _, err := r.GetMessages(
		ctx,
		&GetMessagesOptions{
			ChatID:        options.ChatID,
			MessageIDs:    []int{options.MessageID},
			ConnectionIDs: options.ConnectionIDs,
		},
	)
	if err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return nil, errors.New("no data found for the specified messageID")
	}

	return messages[0], nil
}

func (r *MongoRepository) GetMessages(ctx context.Context, options *GetMessagesOptions) ([]*telego.Message, *PaginationAnswer, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	matchConditions := bson.D{
		{Key: "chat_id", Value: options.ChatID},
		{Key: "business_connection", Value: bson.D{{Key: "$in", Value: options.ConnectionIDs}}},
	}

	orConditions := []bson.D{}
	if len(options.MessageIDs) > 0 {
		orConditions = append(orConditions, bson.D{{Key: "message_id", Value: bson.D{{Key: "$in", Value: options.MessageIDs}}}})
	}

	if len(orConditions) > 0 {
		matchConditions = append(matchConditions, bson.E{Key: "$or", Value: orConditions})
	} else {
		matchConditions = append(matchConditions, bson.E{Key: "$or", Value: bson.A{bson.D{{Key: "_id", Value: nil}}}}) // Matches no documents
	}

	pipeline := mongo.Pipeline{}
	pipeline = append(pipeline, bson.D{{Key: "$match", Value: matchConditions}})

	if options.WithEdits {
		pipeline = append(pipeline, bson.D{{Key: "$sort", Value: bson.D{
			{Key: "message_id", Value: 1},
			{Key: "date", Value: -1}, // 1
			{Key: "edit_date", Value: -1},
		}}})
	} else {
		pipeline = append(
			pipeline,
			bson.D{{Key: "$sort", Value: bson.D{
				{Key: "message_id", Value: 1},
				{Key: "edit_date", Value: -1},
				{Key: "date", Value: -1},
			}}},
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$message_id"},
				{Key: "doc", Value: bson.D{{Key: "$first", Value: "$$ROOT"}}},
			}}},
			bson.D{{Key: "$replaceRoot", Value: bson.D{{Key: "newRoot", Value: "$doc"}}}},
			bson.D{{Key: "$sort", Value: bson.D{{Key: "message_id", Value: 1}}}},
		)
	}

	if options.Offset > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$skip", Value: options.Offset}})
	}

	if options.Limit > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$limit", Value: options.Limit + 1}})
	}

	cursor, err := r.telegramMessages.Aggregate(ctxTimeout, pipeline)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to aggregate messages: %w", err)
	}
	defer cursor.Close(ctxTimeout)

	var messages []*telego.Message
	for cursor.Next(ctxTimeout) {
		var resultDoc internalMessage
		if err := cursor.Decode(&resultDoc); err != nil {
			return nil, nil, fmt.Errorf("failed to decode message document: %w", err)
		}

		var msg telego.Message
		if err := json.Unmarshal([]byte(resultDoc.Json), &msg); err != nil {
			log.Warn().Err(err).Int("retrieved_message_id", resultDoc.MessageID).Msg("failed to unmarshal message JSON")
			continue
		}
		messages = append(messages, &msg)
	}

	if err := cursor.Err(); err != nil {
		return nil, nil, fmt.Errorf("cursor error after iterating messages: %w", err)
	}

	pagination := &PaginationAnswer{
		Backward: options.Offset > 0,
	}
	if options.Limit > 0 {
		messagesLen := len(messages)
		if messagesLen > options.Limit {
			pagination.Forward = true
			messages = messages[:messagesLen-1]
		} else {
			pagination.Forward = false
		}
	}

	return messages, pagination, nil
}
