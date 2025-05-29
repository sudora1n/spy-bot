package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type User struct {
	ID           int64  `bson:"_id"`
	SendMessages bool   `bson:"creator_send_messages"`
	LanguageCode string `bson:"language_code"`
	CreatedAt    int64  `bson:"created_at"`
}

func (r *MongoRepository) UpdateUserSendMessages(ctx context.Context, userId int64, sendMessages bool) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": userId}
	update := bson.M{
		"$set": bson.M{
			"creator_send_messages": sendMessages,
		},
	}
	_, err := r.users.UpdateOne(ctx, filter, update)
	return err
}

func (r *MongoRepository) UpdateUserLanguage(ctx context.Context, userId int64, languageCode string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": userId}
	update := bson.M{
		"$set": bson.M{
			"language_code": languageCode,
		},
	}
	_, err := r.users.UpdateOne(ctx, filter, update)
	return err
}

func (r *MongoRepository) UpdateUser(
	ctx context.Context,
	userId int64,
	languageCode string,
	sendMessage bool,
) (new bool, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": userId}

	setFields := bson.M{}
	if sendMessage {
		setFields["send_messages"] = true
	}

	update := bson.M{
		"$set": setFields,
		"$setOnInsert": bson.M{
			"created_at":    time.Now().Unix(),
			"language_code": languageCode,
		},
	}

	res, err := r.users.UpdateOne(
		ctx,
		filter,
		update,
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return false, err
	}

	new = (res.MatchedCount == 0 && res.UpsertedCount == 1)
	return new, nil
}

func (r *MongoRepository) FindUser(
	ctx context.Context,
	userId int64,
) (*User, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": userId}

	var user User
	if err := r.users.FindOne(ctx, filter).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}
