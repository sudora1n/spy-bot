package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/mymmrac/telego"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type IUser struct {
	User    User     `bson:"user"`
	BotUser *BotUser `bson:"bot_user,omitempty"`
}

type UserSettings struct {
	ShowMyEdits        bool `bson:"show_my_edits"`        // need false default
	ShowPartnerEdits   bool `bson:"show_partner_edits"`   // need true default
	ShowMyDeleted      bool `bson:"show_my_deleted"`      // need true default
	ShowPartnerDeleted bool `bson:"show_partner_deleted"` // need true default
}

type User struct {
	ID int64 `bson:"_id"`

	LanguageCode string        `bson:"language_code"`
	Settings     *UserSettings `bson:"settings"`

	CreatedAt int64 `bson:"created_at"`
}

type BotUserBusinessConnection struct {
	ID       string                    `bson:"id"`
	Rights   *telego.BusinessBotRights `bson:"rights,omitempty"`
	Enabled  bool                      `bson:"enabled"`
	Unixtime int64                     `bson:"date"`
}

type BotUser struct {
	InternalID          int64                       `bson:"_id"`
	BusinessConnections []BotUserBusinessConnection `bson:"business_connections"`
	SendMessages        bool                        `bson:"send_messages"`

	UserID    int64 `bson:"user_id"`
	BotID     int64 `bson:"bot_id"`
	CreatedAt int64 `bson:"created_at"`
}

func (b *BotUser) GetUserCurrentConnection() *BotUserBusinessConnection {
	var latestConnection *BotUserBusinessConnection
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
					"business_connections": BotUserBusinessConnection{
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

func (r *MongoRepository) UpdateUserSettings(ctx context.Context, userID int64, data *UserSettings) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": userID}
	update := bson.M{
		"$set": bson.M{
			"settings.show_my_edits":        data.ShowMyEdits,
			"settings.show_partner_edits":   data.ShowPartnerEdits,
			"settings.show_my_deleted":      data.ShowMyDeleted,
			"settings.show_partner_deleted": data.ShowPartnerDeleted,
		},
	}

	_, err := r.users.UpdateOne(ctx, filter, update)
	return err
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

func (r *MongoRepository) FindBotUser(
	ctx context.Context,
	userId int64,
	botId int64,
) (*BotUser, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"user_id": userId, "bot_ud": botId}

	var botUser BotUser
	if err := r.users.FindOne(ctx, filter).Decode(&botUser); err != nil {
		return nil, err
	}
	return &botUser, nil
}

func (r *MongoRepository) ListIUsers(
	ctx context.Context,
	botID int64,
	page int64,
	pageSize int64,
) ([]IUser, error) {
	if page < 1 || pageSize < 1 {
		return nil, fmt.Errorf("invalid pagination params: page=%d, pageSize=%d", page, pageSize)
	}

	skip := (page - 1) * pageSize

	pipeline := []bson.M{
		{
			"$sort": bson.M{"_id": 1},
		},
		{
			"$skip": skip,
		},
		{
			"$limit": pageSize,
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
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cursor, err := r.users.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []IUser
	for cursor.Next(ctx) {
		var iUser IUser
		if err := cursor.Decode(&iUser); err != nil {
			return nil, err
		}
		results = append(results, iUser)
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func (r *MongoRepository) CreateUser(
	ctx context.Context,
	userId int64,
	languageCode string,
	fromCreator bool,
) (err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	insert := bson.M{
		"$set": bson.M{
			"_id": userId,
			"settings": bson.M{
				"show_my_edits":        false,
				"show_partner_edits":   true,
				"show_my_deleted":      true,
				"show_partner_deleted": true,
			},
			"language_code": languageCode,
			"created_at":    time.Now().Unix(),
		},
	}

	if fromCreator {
		insert["creator_send_messages"] = true
	}

	_, err = r.users.InsertOne(ctx, insert)
	if err != nil {
		return err
	}

	return nil
}

func (r *MongoRepository) CreateBotUser(
	ctx context.Context,
	userId int64,
	botID int64,
) (err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_id, err := r.GetNextSequence(ctx, r.botUsers.Name())
	if err != nil {
		return err
	}

	insert := bson.M{
		"$set": bson.M{
			"_id":           _id.Value,
			"user_id":       userId,
			"bot_id":        botID,
			"send_messages": true,
			"created_at":    time.Now().Unix(),
		},
	}

	_, err = r.botUsers.InsertOne(ctx, insert)
	if err != nil {
		return err
	}

	return nil
}
