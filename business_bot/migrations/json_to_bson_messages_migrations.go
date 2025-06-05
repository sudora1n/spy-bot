package migrations

import (
	"context"
	"encoding/json"
	"fmt"
	custom_registry "ssuspy-bot/repository/bson_custon_registry"
	"time"

	"github.com/mymmrac/telego"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type iOld struct {
	ID   int64  `bson:"_id"`
	Json string `bson:"json"`
}

func DoJsonToBsonMessagesMigrate(ctx context.Context, db *mongo.Database, registry *custom_registry.CustomRegistry) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	oldColl := db.Collection("telegram_messages")
	newColl := db.Collection("telegram_messages_v2")

	cursor, err := oldColl.Find(ctx, bson.D{})
	if err != nil {
		return fmt.Errorf("error find old messages: %v", err)
	}
	defer cursor.Close(ctx)

	var count int
	for cursor.Next(ctx) {
		var old iOld

		if err := cursor.Decode(&old); err != nil {
			log.Warn().Err(err).Msg("err decode old doc")
			continue
		}

		var msg telego.Message
		if err = json.Unmarshal([]byte(old.Json), &msg); err != nil {
			log.Warn().Err(err).Int64("internalID", old.ID).Msg("unmarshal JSON to telego.Message")
			continue
		}

		rawBytes, err := registry.SaveMessage(&msg)
		if err != nil {
			log.Warn().Err(err).Int64("internalID", old.ID).Msg("fail save message via registry")
			continue
		}

		newDoc := bson.D{
			{Key: "_id", Value: old.ID},
			{Key: "message", Value: bson.Raw(rawBytes)},
		}
		if _, err := newColl.InsertOne(ctx, newDoc); err != nil {
			log.Warn().Err(err).Int64("internalID", old.ID).Msg("fail insert message")
			continue
		}

		count++
		if count%500 == 0 {
			log.Info().Int("count", count).Msg("migrated messages")
		}
	}
	if err := cursor.Err(); err != nil {
		log.Fatal().Err(err).Msg("cursor error")
	}

	log.Info().Msg("migration done")
	return nil
}
