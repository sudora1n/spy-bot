package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"ssuspy-bot/config"
	"ssuspy-bot/consts"
)

type MongoRepository struct {
	client              *mongo.Client
	telegramMessages    *mongo.Collection
	users               *mongo.Collection
	callbackDataDeleted *mongo.Collection
	callbackDataEdited  *mongo.Collection
	filesExists         *mongo.Collection
	chatResolve         *mongo.Collection
	bots                *mongo.Collection
	botUsers            *mongo.Collection
	counters            *mongo.Collection
}

type Sequence struct {
	Value int64 `bson:"value"`
}

func NewMongoRepository(cfg *config.MongoConfig) (*MongoRepository, error) {
	uri := cfg.BuildMongoURI()

	clientOptions := options.Client().ApplyURI(uri)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}

	telegramMessages := client.Database(cfg.Database).Collection("telegram_messages")
	userCollection := client.Database(cfg.Database).Collection("users")
	callbackDataDeletedCollection := client.Database(cfg.Database).Collection("callback_data_deleted")
	idxTTLMonth := mongo.IndexModel{
		Keys: bson.D{
			{Key: "created_at", Value: 1},
		},
		Options: options.Index().SetExpireAfterSeconds(consts.MonthInSeconds),
	}
	_, err = callbackDataDeletedCollection.Indexes().CreateOne(ctx, idxTTLMonth)
	if err != nil {
		return nil, err
	}
	callbackDataEditedCollection := client.Database(cfg.Database).Collection("callback_data_edited")
	_, err = callbackDataEditedCollection.Indexes().CreateOne(ctx, idxTTLMonth)
	if err != nil {
		return nil, err
	}
	filesExistsCollection := client.Database(cfg.Database).Collection("files_exists")
	idxModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "fileId", Value: 1},
			{Key: "userId", Value: 1},
			{Key: "chatId", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}
	_, err = filesExistsCollection.Indexes().CreateOne(ctx, idxModel)
	if err != nil {
		return nil, err
	}

	chatResolveCollection := client.Database(cfg.Database).Collection("chats_resolve")
	botsCollection := client.Database(cfg.Database).Collection("bots")
	botUsersCollection := client.Database(cfg.Database).Collection("bot_users")
	idxModel = mongo.IndexModel{
		Keys: bson.D{
			{Key: "bot_id", Value: 1},
			{Key: "userId", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}
	_, err = botUsersCollection.Indexes().CreateOne(ctx, idxModel)

	countersCollection := client.Database(cfg.Database).Collection("counters")

	return &MongoRepository{
		client:              client,
		telegramMessages:    telegramMessages,
		users:               userCollection,
		callbackDataDeleted: callbackDataDeletedCollection,
		callbackDataEdited:  callbackDataEditedCollection,
		filesExists:         filesExistsCollection,
		chatResolve:         chatResolveCollection,
		bots:                botsCollection,
		botUsers:            botUsersCollection,
		counters:            countersCollection,
	}, nil
}

func (r *MongoRepository) GetNextSequence(ctx context.Context, name string) (*Sequence, error) {
	filter := bson.M{"_id": name}
	update := bson.M{"$inc": bson.M{"value": 1}}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)

	var result Sequence
	err := r.counters.FindOneAndUpdate(ctx, filter, update, opts).Decode(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *MongoRepository) Disconnect(ctx context.Context) error {
	if r.client == nil {
		return nil
	}
	return r.client.Disconnect(ctx)
}
