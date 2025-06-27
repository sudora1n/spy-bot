package repository

import (
	"context"
	"time"

	"github.com/mymmrac/telego"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type IUser struct {
	User    User     `bson:"user"`
	BotUser *BotUser `bson:"bot_user,omitempty"`
}

type User struct {
	ID int64 `bson:"_id"`

	LanguageCode string `bson:"language_code"`
	CreatedAt    int64  `bson:"created_at"`
}

type BusinessConnection struct {
	ID       string                    `bson:"id"`
	Rights   *telego.BusinessBotRights `bson:"rights,omitempty"`
	Enabled  bool                      `bson:"enabled"`
	Unixtime int64                     `bson:"date"`
}

type BotUser struct {
	InternalID          int64                `bson:"_id"`
	BusinessConnections []BusinessConnection `bson:"business_connections"`
	SendMessages        bool                 `bson:"send_messages"`

	UserID    int64 `bson:"user_id"`
	BotID     int64 `bson:"bot_id"`
	CreatedAt int64 `bson:"created_at"`
}

func (b *BotUser) GetUserCurrentConnection() *BusinessConnection {
	var latestConnection *BusinessConnection
	for i := range b.BusinessConnections {
		conn := &b.BusinessConnections[i]
		if conn.Enabled {
			if latestConnection == nil || conn.Unixtime > latestConnection.Unixtime {
				latestConnection = conn
			}
		}
	}
	return latestConnection
}

func (b *BotUser) GetUserCurrentConnectionIDs() []string {
	connectionIDs := make([]string, len(b.BusinessConnections))
	for i, connection := range b.BusinessConnections {
		connectionIDs[i] = connection.ID
	}

	return connectionIDs
}

func (r *MongoRepository) updateUser(
	ctx context.Context,
	userId int64,
	languageCode string,
) (new bool, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": userId}
	update := bson.M{
		"$setOnInsert": bson.M{
			"language_code": languageCode,
			"created_at":    time.Now().Unix(),
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

func (r *MongoRepository) updateBotUser(
	ctx context.Context,
	userId int64,
	botID int64,
	sendMessage bool,
) (new bool, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{
		"user_id": userId,
		"bot_id":  botID,
	}

	setFields := bson.M{}
	if sendMessage {
		setFields["send_messages"] = true
	}

	_id, err := r.GetNextSequence(ctx, r.botUsers.Name())
	if err != nil {
		return false, err
	}

	update := bson.M{
		"$set": setFields,
		"$setOnInsert": bson.M{
			"_id":        _id.Value,
			"created_at": time.Now().Unix(),
		},
	}

	res, err := r.botUsers.UpdateOne(
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

func (r *MongoRepository) UpdateBotUserConnection(ctx context.Context, connection *telego.BusinessConnection, botID int64) (isUpdated bool, err error) {
	currentTime := time.Now().Unix()

	if connection.IsEnabled {
		filter := bson.M{
			"user_id":                 connection.User.ID,
			"bot_id":                  botID,
			"business_connections.id": connection.ID,
		}

		updateFields := bson.M{
			"business_connections.$.date":    currentTime,
			"business_connections.$.enabled": true,
		}

		if connection.Rights != nil {
			updateFields["business_connections.$.rights"] = connection.Rights
		}

		update := bson.M{
			"$set": updateFields,
		}

		result, err := r.botUsers.UpdateOne(ctx, filter, update)
		if err != nil {
			return false, err
		}
		if result.ModifiedCount > 0 {
			return true, nil
		} else {
			_id, err := r.GetNextSequence(ctx, r.botUsers.Name())
			if err != nil {
				return false, err
			}

			update := bson.M{
				"$push": bson.M{
					"business_connections": BusinessConnection{
						ID:       connection.ID,
						Enabled:  true,
						Unixtime: currentTime,
						Rights:   connection.Rights,
					},
				},
				"$setOnInsert": bson.M{
					"_id":        _id.Value,
					"user_id":    connection.User.ID,
					"bot_id":     botID,
					"created_at": currentTime,
				},
			}

			_, err = r.botUsers.UpdateOne(
				ctx,
				bson.M{
					"user_id": connection.User.ID,
					"bot_id":  botID,
				},
				update,
				options.Update().SetUpsert(true),
			)
			if err != nil {
				return false, err
			}

			return false, nil
		}
	} else {
		filter := bson.M{
			"user_id":                 connection.User.ID,
			"bot_id":                  botID,
			"business_connections.id": connection.ID,
		}

		updateFields := bson.M{
			"business_connections.$.enabled": false,
		}

		update := bson.M{
			"$set": updateFields,
		}
		result, err := r.botUsers.UpdateOne(ctx, filter, update)
		if err != nil {
			return false, err
		}

		return result.ModifiedCount > 0, nil
	}
}

func (r *MongoRepository) UpdateBotUserSendMessages(ctx context.Context, userId int64, botID int64, sendMessages bool) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{
		"user_id": userId,
		"bot_id":  botID,
	}
	update := bson.M{
		"$set": bson.M{
			"send_messages": sendMessages,
		},
	}
	_, err := r.botUsers.UpdateOne(ctx, filter, update)
	return err
}

func (r *MongoRepository) FindBotUser(
	ctx context.Context,
	userId int64,
	botID int64,
) (*BotUser, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{
		"user_id": userId,
		"bot_id":  botID,
	}

	var botUser BotUser
	if err := r.botUsers.FindOne(ctx, filter).Decode(&botUser); err != nil {
		return nil, err
	}
	return &botUser, nil
}

func (r *MongoRepository) FindIUserByConnectionID(
	ctx context.Context,
	businessConnectionID string,
	botID int64,
) (*IUser, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	pipeline := []bson.M{
		{
			"$match": bson.M{
				"bot_id": botID,
				"business_connections": bson.M{
					"$elemMatch": bson.M{
						"id":      businessConnectionID,
						"enabled": true,
					},
				},
				"send_messages": true,
			},
		},
		{
			"$lookup": bson.M{
				"from":         "users",
				"localField":   "user_id",
				"foreignField": "_id",
				"as":           "user_data",
			},
		},
		{
			"$unwind": "$user_data",
		},
		{"$project": bson.M{
			"user":     "$user_data",
			"bot_user": "$$ROOT",
		}},
		{"$limit": 1},
	}

	cursor, err := r.botUsers.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var result IUser
	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		return &result, nil
	}

	return nil, mongo.ErrNoDocuments
}

func (r *MongoRepository) FindIUserByID(
	ctx context.Context,
	userID int64,
	botID int64,
) (*IUser, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	pipeline := []bson.M{
		{
			"$match": bson.M{
				"_id": userID,
			},
		},
		{
			"$lookup": bson.M{
				"from":         "bot_users",
				"localField":   "_id",
				"foreignField": "user_id",
				"as":           "bot_users",
			},
		},
		{
			"$unwind": bson.M{
				"path":                       "$bot_users",
				"preserveNullAndEmptyArrays": true,
			},
		},
		{
			"$match": bson.M{
				"bot_users.bot_id": botID,
			},
		},
		{
			"$project": bson.M{
				"user":     "$$ROOT",
				"bot_user": "$bot_users",
			},
		},
		{"$limit": 1},
	}

	cursor, err := r.users.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var result IUser
	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		return &result, nil
	}

	return nil, mongo.ErrNoDocuments
}

func (r *MongoRepository) FindOrCreateIUser(
	ctx context.Context,
	userId int64,
	botID int64,
	languageCode string,
) (*IUser, bool, error) {
	iUser, err := r.FindIUserByID(ctx, userId, botID)
	if err == nil {
		return iUser, false, nil
	}

	if err != mongo.ErrNoDocuments {
		log.Debug().Int64("userID", userId).Int64("botID", botID).Err(err).Msg("err find user")
		return nil, false, err
	}

	_, err = r.updateUser(ctx, userId, languageCode)
	if err != nil {
		log.Debug().Int64("userID", userId).Int64("botID", botID).Err(err).Msg("err update user")
		return nil, false, err
	}

	isNew, err := r.updateBotUser(ctx, userId, botID, false)
	if err != nil {
		log.Debug().Int64("userID", userId).Int64("botID", botID).Err(err).Msg("err update bot user")
		return nil, false, err
	}

	iUser, err = r.FindIUserByID(ctx, userId, botID)
	if err != nil {
		log.Debug().Int64("userID", userId).Int64("botID", botID).Err(err).Msg("err find user by id")
		return nil, false, err
	}

	return iUser, isNew, nil
}
