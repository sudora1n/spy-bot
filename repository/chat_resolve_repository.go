package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

type ChatResolve struct {
	ID   int64  `bson:"_id"`
	Name string `bson:"name"`
}

func (r *MongoRepository) UpdateChatName(ctx context.Context, chatID int64, name string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{
		"_id": chatID,
	}

	update := bson.M{
		"$set": bson.M{
			"name": name,
		},
	}

	_, err := r.chatResolve.UpdateOne(ctx, filter, update)
	return err
}

func (r *MongoRepository) FindChatName(
	ctx context.Context,
	chatID int64,
) (*ChatResolve, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": chatID}

	var chat ChatResolve
	if err := r.chatResolve.FindOne(ctx, filter).Decode(&chat); err != nil {
		return nil, err
	}
	return &chat, nil
}
