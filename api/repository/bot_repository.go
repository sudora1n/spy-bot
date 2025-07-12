package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Bot struct {
	ID int64 `bson:"_id"`

	Username    string `bson:"username"`
	SecretToken string `bson:"secret_token"`

	UserID    int64     `bson:"user_id"`
	CreatedAt time.Time `bson:"created_at"`
}

type FindBotWithUserCountsResult struct {
	Bot                Bot   `bson:"bot"`
	TotalUsers         int64 `bson:"totalUsers"`
	TotalBusinessUsers int64 `bson:"totalBusinessUsers"`
}

func (r *MongoRepository) LenBots(
	ctx context.Context,
	userId int64,
) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"user_id": userId}
	return r.bots.CountDocuments(ctx, filter)
}

func (r *MongoRepository) FindBotWithUserCounts(
	ctx context.Context,
	userId int64,
	botId int64,
) (*FindBotWithUserCountsResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.D{
			{Key: "_id", Value: botId},
			{Key: "user_id", Value: userId},
		}}},
		bson.D{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "bot_users"},
			{Key: "let", Value: bson.D{{Key: "botId", Value: "$_id"}}},
			{Key: "pipeline", Value: mongo.Pipeline{
				bson.D{{Key: "$match", Value: bson.D{
					{Key: "$expr", Value: bson.D{{Key: "$eq", Value: bson.A{"$bot_id", "$$botId"}}}},
				}}},
				bson.D{{Key: "$group", Value: bson.D{
					{Key: "_id", Value: nil},
					{Key: "totalUsers", Value: bson.D{{Key: "$sum", Value: 1}}},
					{Key: "totalBusinessUsers", Value: bson.D{{Key: "$sum", Value: bson.D{
						{Key: "$cond", Value: bson.A{
							// условие: есть хоть одно enabled=true в business_connections
							bson.D{{Key: "$gt", Value: bson.A{
								bson.D{{Key: "$size", Value: bson.D{
									{Key: "$filter", Value: bson.D{
										{Key: "input", Value: bson.D{{Key: "$ifNull", Value: bson.A{"$business_connections", bson.A{}}}}},
										{Key: "as", Value: "bc"},
										{Key: "cond", Value: bson.D{{Key: "$eq", Value: bson.A{"$$bc.enabled", true}}}},
									}},
								}}},
								0,
							}}},
							1,
							0,
						}},
					}}}},
				}}},
			}},
			{Key: "as", Value: "counts"},
		}}},

		bson.D{{Key: "$unwind", Value: "$counts"}},
		bson.D{{Key: "$addFields", Value: bson.D{
			{Key: "totalUsers", Value: "$counts.totalUsers"},
			{Key: "totalBusinessUsers", Value: "$counts.totalBusinessUsers"},
		}}},
		bson.D{{Key: "$project", Value: bson.D{
			{Key: "bot", Value: "$$ROOT"},
			{Key: "totalUsers", Value: "$totalUsers"},
			{Key: "totalBusinessUsers", Value: "$totalBusinessUsers"},
			{Key: "counts", Value: 0},
		}}},
	}

	cursor, err := r.bots.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var result FindBotWithUserCountsResult
	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		return &result, nil
	}

	return nil, mongo.ErrNoDocuments
}

func (r *MongoRepository) FindBotByToken(
	ctx context.Context,
	userId int64,
	token string,
) (*Bot, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"user_id": userId, "secret_token": token}

	var bot Bot
	if err := r.bots.FindOne(ctx, filter).Decode(&bot); err != nil {
		return nil, err
	}
	return &bot, nil
}

func (r *MongoRepository) FindBots(
	ctx context.Context,
	userId int64,
) ([]Bot, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"user_id": userId}

	cursor, err := r.bots.Find(ctx, filter)
	if err != nil {
		return nil, err
	}

	var bots []Bot
	if err := cursor.All(ctx, &bots); err != nil {
		return nil, err
	}

	return bots, nil
}

func (r *MongoRepository) InsertBot(
	ctx context.Context,
	botID int64,
	userId int64,
	token string,
	username string,
) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	bot := &Bot{
		ID:          botID,
		SecretToken: token,
		UserID:      userId,
		CreatedAt:   time.Now(),
		Username:    username,
	}

	_, err := r.bots.InsertOne(ctx, bot)
	return err
}

func (r *MongoRepository) RemoveBot(
	ctx context.Context,
	userId int64,
	botID int64,
) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"user_id": userId, "_id": botID}

	_, err := r.bots.DeleteOne(ctx, filter)
	return err
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
	defer cursor.Close(ctx)
	if err != nil {
		return nil, err
	}

	var bots []Bot
	if err := cursor.All(ctx, &bots); err != nil {
		return nil, err
	}

	return bots, nil
}
