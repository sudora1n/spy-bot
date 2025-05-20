package repository

import (
	"context"
	"fmt"
	"ssuspy-bot/config"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (r *MongoRepository) CreateFileIfNotExists(ctx context.Context, fileID string, userID int64, chatID int64) (bool, error) {
	if config.Config.DevMode {
		return true, nil
	}

	filter := bson.M{
		"fileId": fileID,
		"userId": userID,
		"chatId": chatID,
	}

	id, err := r.GetNextSequence(ctx, r.filesExists.Name())
	if err != nil {
		return false, fmt.Errorf("failed to get next seq: %w", err)
	}

	// надо переписать на findone+updateone чтобы не задействовать counters просто так
	update := bson.M{
		"$setOnInsert": bson.M{
			"_id":       id,
			"fileId":    fileID,
			"userId":    userID,
			"chatId":    chatID,
			"createdAt": time.Now(),
		},
	}

	opts := options.Update().SetUpsert(true)

	res, err := r.filesExists.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return false, err
	}

	if res.UpsertedCount == 1 {
		return true, nil
	}

	return false, nil
}
