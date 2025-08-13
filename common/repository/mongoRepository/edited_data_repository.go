package mongoRepository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DataEdited struct {
	ID            int64 `bson:"_id"`
	MessageID     int   `bson:"message_id"`
	UserID        int64 `bson:"user_id"`
	OldDate       int64 `bson:"old_date"`
	OldDateIsEdit bool  `bson:"old_date_is_edit"`
	NewDate       int64 `bson:"new_date"`

	CreatedAt time.Time `bson:"created_at"`
}

type SetDataEditedOptions struct {
	MessageID     int
	UserID        int64
	OldDate       int64
	OldDateIsEdit bool
	NewDate       int64
}

func (r *MongoRepository) SetDataEdited(ctx context.Context, options *SetDataEditedOptions) (primitive.ObjectID, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := DataEdited{
		MessageID:     options.MessageID,
		UserID:        options.UserID,
		OldDate:       options.OldDate,
		OldDateIsEdit: options.OldDateIsEdit,
		NewDate:       options.NewDate,
		CreatedAt:     time.Now(),
	}

	res, err := r.callbackDataEdited.InsertOne(ctx, row)

	objectID, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return primitive.NilObjectID, fmt.Errorf("failed to convert to objectid")
	}

	return objectID, err
}

func (r *MongoRepository) GetDataEdited(ctx context.Context, userId int64, id primitive.ObjectID) (*DataEdited, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": id, "user_id": userId}

	var data DataEdited
	if err := r.callbackDataEdited.FindOne(ctx, filter).Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}
