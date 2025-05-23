package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

type DataDeleted struct {
	ID         int64 `bson:"_id"`
	MessageIDs []int `bson:"message_ids"`
	UserID     int64 `bson:"user_id"`

	CreatedAt time.Time `bson:"created_at"`
}

func (r *MongoRepository) SetDataDeleted(ctx context.Context, userID int64, messageIDs []int) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	id, err := r.GetNextSequence(ctx, r.callbackDataDeleted.Name())
	if err != nil {
		return 0, fmt.Errorf("failed get next seq: %w", err)
	}

	row := DataDeleted{
		ID:         id.Value,
		MessageIDs: messageIDs,
		UserID:     userID,
		CreatedAt:  time.Now(),
	}

	_, err = r.callbackDataDeleted.InsertOne(ctx, row)
	return id.Value, err
}

func (r *MongoRepository) GetDataDeleted(ctx context.Context, userId int64, id int64) (*DataDeleted, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": id, "user_id": userId}

	var data DataDeleted
	if err := r.callbackDataDeleted.FindOne(ctx, filter).Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}
