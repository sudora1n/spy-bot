package mongoRepository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DataDeleted struct {
	ID            int64 `bson:"_id"`
	MessageIDs    []int `bson:"message_ids"`
	UserID        int64 `bson:"user_id"`
	MessagesCount uint8 `bson:"messages_count"`
	FilesCount    uint8 `bson:"files_count"`

	CreatedAt time.Time `bson:"created_at"`
}

func (r *MongoRepository) SetDataDeleted(ctx context.Context, userID int64, messageIDs []int, messagesCount uint8, filesCount uint8) (primitive.ObjectID, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := DataDeleted{
		MessageIDs:    messageIDs,
		UserID:        userID,
		MessagesCount: messagesCount,
		FilesCount:    filesCount,
		CreatedAt:     time.Now(),
	}

	res, err := r.callbackDataDeleted.InsertOne(ctx, row)

	objectID, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return primitive.NilObjectID, fmt.Errorf("failed to convert to objectid")
	}

	return objectID, err
}

func (r *MongoRepository) GetDataDeleted(ctx context.Context, userId int64, id primitive.ObjectID) (*DataDeleted, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": id, "user_id": userId}

	var data DataDeleted
	if err := r.callbackDataDeleted.FindOne(ctx, filter).Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}
