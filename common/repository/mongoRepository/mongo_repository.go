package mongoRepository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	custom_registry "ssuspy-common/bson"
	"ssuspy-common/consts"
	"ssuspy-common/types"
)

type MongoRepository struct {
	client              *mongo.Client
	db                  *mongo.Database
	telegramMessages    *mongo.Collection
	users               *mongo.Collection
	callbackDataDeleted *mongo.Collection
	callbackDataEdited  *mongo.Collection
	filesExists         *mongo.Collection
	chats               *mongo.Collection
	peers               *mongo.Collection
	bots                *mongo.Collection
	botUsers            *mongo.Collection
	counters            *mongo.Collection
	migrations          *mongo.Collection

	customRegistry *custom_registry.CustomRegistry
}

type Sequence struct {
	Value int64 `bson:"value"`
}

func NewMongoRepository(cfg *types.MongoConfig) (*MongoRepository, error) {
	ctx := context.Background()

	uri := cfg.BuildMongoURI()

	customRegistry := custom_registry.CreateCustomRegistry()

	clientOptions := options.Client().
		SetRegistry(customRegistry.Registry).
		ApplyURI(uri)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}

	db := client.Database(cfg.Database)

	telegramMessages := db.Collection("telegram_messages_v2")
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "message.chat.id", Value: 1},
			},
			Options: options.Index().SetName("ChatId"),
		},
		{
			Keys: bson.D{
				{Key: "message.chat.id", Value: 1},
				{Key: "message.business_connection_id", Value: 1},
				{Key: "message.message_id", Value: 1},
				{Key: "message.edit_date", Value: -1},
				{Key: "message.date", Value: -1},
			},
			Options: options.Index().SetName("ChatConn_MsgId_EditDate_Date"),
		},
		{
			Keys: bson.D{
				{Key: "message.chat.id", Value: 1},
				{Key: "message.business_connection_id", Value: 1},
				{Key: "message.message_id", Value: 1},
			},
			Options: options.Index().SetName("ChatConn_MsgId"),
		},
	}
	_, err = telegramMessages.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return nil, err
	}

	userCollection := db.Collection("users")
	callbackDataDeletedCollection := db.Collection("callback_data_deleted")
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
	callbackDataEditedCollection := db.Collection("callback_data_edited")
	_, err = callbackDataEditedCollection.Indexes().CreateOne(ctx, idxTTLMonth)
	if err != nil {
		return nil, err
	}
	filesExistsCollection := db.Collection("files_exists")
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

	chatsCollection := db.Collection("chats")
	peersCollection := db.Collection("peers")
	botsCollection := db.Collection("bots")
	idxModel = mongo.IndexModel{
		Keys: bson.D{
			{Key: "user_id", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}
	_, err = botsCollection.Indexes().CreateOne(ctx, idxModel)

	botUsersCollection := db.Collection("bot_users")
	idxModel = mongo.IndexModel{
		Keys: bson.D{
			{Key: "bot_id", Value: 1},
			{Key: "user_id", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}
	_, err = botUsersCollection.Indexes().CreateOne(ctx, idxModel)

	countersCollection := db.Collection("counters")
	migrationsCollection := db.Collection("migrations")

	repository := MongoRepository{
		client:              client,
		db:                  db,
		telegramMessages:    telegramMessages,
		users:               userCollection,
		callbackDataDeleted: callbackDataDeletedCollection,
		callbackDataEdited:  callbackDataEditedCollection,
		filesExists:         filesExistsCollection,
		chats:               chatsCollection,
		peers:               peersCollection,
		bots:                botsCollection,
		botUsers:            botUsersCollection,
		counters:            countersCollection,
		migrations:          migrationsCollection,
		customRegistry:      customRegistry,
	}

	return &repository, nil
}

func (r *MongoRepository) GetDb() *mongo.Database {
	return r.db
}

func (r *MongoRepository) Disconnect(ctx context.Context) error {
	if r.client == nil {
		return nil
	}
	return r.client.Disconnect(ctx)
}
