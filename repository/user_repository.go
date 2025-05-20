package repository

import (
	"context"
	"time"

	"github.com/mymmrac/telego"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type BusinessConnection struct {
	ID       string `bson:"id"`
	Enabled  bool   `bson:"enabled"`
	Unixtime int64  `bson:"date"`
}

type User struct {
	ID                  int64                `bson:"_id"`
	BusinessConnections []BusinessConnection `bson:"business_connections"`
	SendMessages        bool                 `bson:"send_messages"`
	LanguageCode        string               `bson:"language_code"`
	CreatedAt           int64                `bson:"created_at"`
}

func (u *User) GetUserCurrentConnection() *BusinessConnection {
	var latestConnection *BusinessConnection
	for i := range u.BusinessConnections {
		conn := &u.BusinessConnections[i]
		if conn.Enabled {
			if latestConnection == nil || conn.Unixtime > latestConnection.Unixtime {
				latestConnection = conn
			}
		}
	}
	return latestConnection
}

func (u *User) GetUserCurrentConnectionIDs() []string {
	connectionIDs := make([]string, len(u.BusinessConnections))
	for i, connection := range u.BusinessConnections {
		connectionIDs[i] = connection.ID
	}

	return connectionIDs
}
func (r *MongoRepository) UpdateUserConnection(ctx context.Context, connection *telego.BusinessConnection) error {
	currentTime := time.Now().Unix()

	if connection.IsEnabled {
		filter := bson.M{
			"_id":                     connection.User.ID,
			"business_connections.id": connection.ID,
		}

		update := bson.M{
			"$set": bson.M{
				"business_connections.$.date":    currentTime,
				"business_connections.$.enabled": true,
			},
		}

		result, err := r.users.UpdateOne(ctx, filter, update)
		if err != nil {
			return err
		}

		if result.ModifiedCount == 0 {
			update := bson.M{
				"$push": bson.M{
					"business_connections": BusinessConnection{
						ID:       connection.ID,
						Enabled:  true,
						Unixtime: currentTime,
					},
				},
			}

			_, err := r.users.UpdateOne(
				ctx,
				bson.M{"_id": connection.User.ID},
				update,
				options.Update().SetUpsert(true),
			)
			if err != nil {
				return err
			}
		}
	} else {
		filter := bson.M{
			"_id":                     connection.User.ID,
			"business_connections.id": connection.ID,
		}

		update := bson.M{
			"$set": bson.M{
				"business_connections.$.enabled": false,
			},
		}

		_, err := r.users.UpdateOne(ctx, filter, update)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *MongoRepository) UpdateUserSendMessages(ctx context.Context, userId int64, sendMessages bool) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": userId}
	update := bson.M{
		"$set": bson.M{
			"send_messages": sendMessages,
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

func (r *MongoRepository) FindUserByConnectionID(ctx context.Context, businessConnectionID string) (*User, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{
		"business_connections": bson.M{
			"$elemMatch": bson.M{
				"id":      businessConnectionID,
				"enabled": true,
			},
		},
		"send_messages": true,
	}

	var user User
	if err := r.users.FindOne(ctx, filter).Decode(&user); err != nil {
		log.Error().Str("businessConnectionID", businessConnectionID).Err(err).Msg("error decoding user")
		return nil, err
	}
	return &user, nil
}
