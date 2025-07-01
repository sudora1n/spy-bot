package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"ssuspy-bot/config"
	"ssuspy-bot/consts"
	"ssuspy-bot/migrations"
	custom_registry "ssuspy-bot/repository/bson_custon_registry"
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
	migrations          *mongo.Collection

	customRegistry *custom_registry.CustomRegistry
}

type Sequence struct {
	Value int64 `bson:"value"`
}

func NewMongoRepository(cfg *config.MongoConfig) (*MongoRepository, error) {
	uri := cfg.BuildMongoURI()

	customRegistry := custom_registry.CreateCustomRegistry()

	clientOptions := options.Client().
		SetRegistry(customRegistry.Registry).
		ApplyURI(uri)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

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

	chatResolveCollection := db.Collection("chats_resolve")
	botsCollection := db.Collection("bots")
	botUsersCollection := db.Collection("bot_users")
	idxModel = mongo.IndexModel{
		Keys: bson.D{
			{Key: "bot_id", Value: 1},
			{Key: "userId", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}
	_, err = botUsersCollection.Indexes().CreateOne(ctx, idxModel)

	countersCollection := db.Collection("counters")
	migrationsCollection := db.Collection("migrations")

	repository := MongoRepository{
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
		migrations:          migrationsCollection,

		customRegistry: customRegistry,
	}

	jsonToBsonMessages := "json_to_bson_messages"
	JsonToBsonMessagesMigrationIsNeeded, err := repository.MigrationIsNeeded(ctx, jsonToBsonMessages)
	if err != nil {
		return nil, err
	}
	if JsonToBsonMessagesMigrationIsNeeded {
		migrations.DoJsonToBsonMessagesMigrate(context.Background(), db, customRegistry)
		if err := repository.ApplyMigration(ctx, jsonToBsonMessages); err != nil {
			return nil, err
		}
	}

	addUserSettings := "add_user_settings"
	AddUserSettingsIsNeeded, err := repository.MigrationIsNeeded(ctx, addUserSettings)
	if err != nil {
		return nil, err
	}
	if AddUserSettingsIsNeeded {
		migrations.DoAddUserSettingsMigrate(context.Background(), repository.users)
		if err := repository.ApplyMigration(ctx, addUserSettings); err != nil {
			return nil, err
		}
	}

	userSettingsShorter := "user_settings_shorter"
	UserSettingsShorterIsNeeded, err := repository.MigrationIsNeeded(ctx, userSettingsShorter)
	if err != nil {
		return nil, err
	}
	if UserSettingsShorterIsNeeded {
		migrations.DoUserSettingsShorterMigrate(context.Background(), repository.users)
		if err := repository.ApplyMigration(ctx, userSettingsShorter); err != nil {
			return nil, err
		}
	}

	return &repository, nil
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
