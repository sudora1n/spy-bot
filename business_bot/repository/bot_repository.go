package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

type Bot struct {
	ID int64 `bson:"_id"`

	Username    string `bson:"username"`
	SecretToken string `bson:"secret_token"`

	UserID    int64     `bson:"user_id"`
	CreatedAt time.Time `bson:"created_at"`
}

func (r *MongoRepository) BotByID(
	ctx context.Context,
	botId int64,
) (*Bot, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": botId}

	var bot Bot
	if err := r.bots.FindOne(ctx, filter).Decode(&bot); err != nil {
		return nil, err
	}
	return &bot, nil
}

func (r *MongoRepository) AllBots(
	ctx context.Context,
) ([]Bot, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{}
	cursor, err := r.bots.Find(ctx, filter)
	if err != nil {
		return nil, err
	}

	var bots []Bot
	if err := cursor.All(ctx, bots); err != nil {
		return nil, err
	}

	return bots, nil
}
