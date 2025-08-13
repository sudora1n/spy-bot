package mongoRepository

import (
	"context"
	"fmt"
	"ssuspy-common/telegram/format"
	"time"

	"github.com/mymmrac/telego"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Chat struct {
	ID                  primitive.ObjectID `bson:"_id"`
	UserID              int64              `bson:"user_id"`
	PeerID              int64              `bson:"peer_id"`
	LastMessageUnixTime int64              `bson:"last_message_unixtime"`
	LastMessageID       int64              `bson:"last_message_id"`
}

type ChatWithPeer struct {
	Chat Chat `bson:",inline"`
	Peer Peer `bson:"peer"`
}

func (r *MongoRepository) syncAndGetChat(ctx context.Context, userID int64, message *telego.Message) (*Chat, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var chat Chat
	err := r.chats.FindOneAndUpdate(
		ctx,
		bson.M{
			"user_id": userID,
			"peer_id": message.Chat.ID,
		},
		bson.M{
			"$set": bson.M{
				"last_message_unixtime": format.ChooseTime(message.EditDate, message.Date),
				"last_message_id":       message.MessageID,
			},
		},
		options.FindOneAndUpdate().
			SetUpsert(true).
			SetReturnDocument(options.After),
	).Decode(&chat)

	return &chat, err
}

func (r *MongoRepository) getChatWithPeer(ctx context.Context, userID int64, peerID int64) (*ChatWithPeer, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"user_id": userID,
			"peer_id": peerID,
		}}},
		{{Key: "$lookup", Value: bson.M{
			"from":         "peers",
			"localField":   "peer_id",
			"foreignField": "_id",
			"as":           "peer",
		}}},
		{{Key: "$unwind", Value: bson.M{
			"path":                       "$peer",
			"preserveNullAndEmptyArrays": true,
		}}},
	}

	cursor, err := r.chats.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregate failed: %w", err)
	}

	var results []ChatWithPeer
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	if len(results) == 0 {
		return nil, nil
	}

	return &results[0], nil
}
