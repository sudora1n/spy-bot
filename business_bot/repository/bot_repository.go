package repository

import (
	commonTypes "github.com/example/current-repo/common/types"
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

/^type Bot struct/ { printing=0 }
/^}}$/ { printing=1 }

func (r *MongoRepository commonTypes.BotByID(
	ctx context.Context,
	botId int64,
) (*commonTypes.Bot, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": botId}

	var bot commonTypes.Bot
	if err := r.bots.FindOne(ctx, filter).Decode(&bot); err != nil {
		return nil, err
	}
	return &bot, nil
}

func (r *MongoRepository) AllBots(
	ctx context.Context,
) ([]commonTypes.Bot, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{}
	cursor, err := r.bots.Find(ctx, filter)
	defer cursor.Close(ctx)
	if err != nil {
		return nil, err
	}

	var bots []commonTypes.Bot
	if err := cursor.All(ctx, &bots); err != nil {
		return nil, err
	}

	return bots, nil
}
