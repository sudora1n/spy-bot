package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
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

func (r *MongoRepository) SetDataEdited(ctx context.Context, options *SetDataEditedOptions) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	id, err := r.GetNextSequence(ctx, r.callbackDataEdited.Name())
	if err != nil {
		return 0, fmt.Errorf("failed get next seq: %w", err)
	}

	row := DataEdited{
		ID:            id.Value,
		MessageID:     options.MessageID,
		UserID:        options.UserID,
		OldDate:       options.OldDate,
		OldDateIsEdit: options.OldDateIsEdit,
		NewDate:       options.NewDate,
		CreatedAt:     time.Now(),
	}

	_, err = r.callbackDataEdited.InsertOne(ctx, row)
	return id.Value, err
}

func (r *MongoRepository) GetDataEdited(ctx context.Context, userId int64, id int64) (*DataEdited, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": id, "user_id": userId}

	var data DataEdited
	if err := r.callbackDataEdited.FindOne(ctx, filter).Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}
