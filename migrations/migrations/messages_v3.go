package migrations

import (
	"context"
	"fmt"
	custom_registry "ssuspy-common/bson"
	"time"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/mymmrac/telego"
)

type BotUser struct {
	InternalID          int64                       `bson:"_id"`
	BusinessConnections []BotUserBusinessConnection `bson:"business_connections"`
	SendMessages        bool                        `bson:"send_messages"`

	UserID    int64 `bson:"user_id"`
	BotID     int64 `bson:"bot_id"`
	CreatedAt int64 `bson:"created_at"`
}

type BotUserBusinessConnection struct {
	ID       string                    `bson:"id"`
	Rights   *telego.BusinessBotRights `bson:"rights,omitempty"`
	Enabled  bool                      `bson:"enabled"`
	Unixtime int64                     `bson:"date"`
}

type internalMessageRaw struct {
	InternalID int64    `bson:"_id"`
	Message    bson.Raw `bson:"message"`
}

type Message struct {
	ID       primitive.ObjectID   `bson:"_id"`
	ChatIDs  []primitive.ObjectID `bson:"chat_ids"`
	Messages []bson.Raw           `bson:"messages"`
}

type Chat struct {
	ID                  primitive.ObjectID `bson:"_id"`
	UserID              int64              `bson:"user_id"` // unique user_id->peer_id
	PeerID              int64              `bson:"peer_id"`
	LastMessageUnixTime int64              `bson:"last_message_unixtime"`
	LastMessageID       int64              `bson:"last_message_id"`
}

type Peer struct {
	ID        int64  `bson:"_id"`
	FirstName string `bson:"first_name"`
	LastName  string `bson:"last_name,omitempty"`
	Username  string `bson:"username,omitempty"`
}

type messageGroup struct {
	UserID    int64
	PeerID    int64
	MessageID int
	Messages  []bson.Raw
	ChatID    primitive.ObjectID
}

func DoMigrateToTelegramMessagesV3(
	ctx context.Context,
	client *mongo.Client,
	customRegistry *custom_registry.CustomRegistry,
	messageCollection *mongo.Collection,
	oldMessageCollection *mongo.Collection,
	usersCollection *mongo.Collection,
	peersCollection *mongo.Collection,
	chatsCollection *mongo.Collection,
	chatsResolveCollection *mongo.Collection,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Группируем сообщения по (user_id, peer_id, message_id)
	messageGroups := make(map[string]*messageGroup)
	peerUpdates := make(map[int64]bson.M)

	cursor, err := oldMessageCollection.Find(ctx, bson.D{})
	if err != nil {
		log.Fatal().Err(err).Msg("failed get old messages in cursor")
	}
	defer cursor.Close(ctx)

	log.Info().Msg("Starting message grouping phase...")

	for cursor.Next(ctx) {
		var old internalMessageRaw
		if err := cursor.Decode(&old); err != nil {
			log.Warn().Err(err).Msg("decode error")
			continue
		}

		var msg telego.Message
		if err := customRegistry.LoadMessage(old.Message, &msg); err != nil {
			log.Warn().Err(err).Msg("fail decode message")
			continue
		}

		var botUser BotUser
		err = usersCollection.FindOne(ctx, bson.M{"business_connections.id": msg.BusinessConnectionID}).Decode(&botUser)
		if err != nil {
			log.Warn().Err(err).Str("BCID", msg.BusinessConnectionID).Msg("bot user not found for BCID")
			continue
		}

		userID, peerID := botUser.UserID, msg.Chat.ID
		messageID := msg.MessageID

		// Сохраняем информацию о peer'е
		peerUpdates[peerID] = bson.M{
			"first_name": msg.Chat.FirstName,
			"last_name":  msg.Chat.LastName,
			"username":   msg.Chat.Username,
		}

		// Очищаем поля сообщения
		msg.BusinessConnectionID = ""
		msg.Chat = telego.Chat{}

		msgBytes, err := customRegistry.SaveMessage(&msg)
		if err != nil {
			log.Error().Err(err).Msg("failed to marshal message to BSON")
			continue
		}
		msgRaw := bson.Raw(msgBytes)

		// Создаем ключ для группировки
		groupKey := fmt.Sprintf("%d_%d_%d", userID, peerID, messageID)

		if _, exists := messageGroups[groupKey]; !exists {
			messageGroups[groupKey] = &messageGroup{
				UserID:    userID,
				PeerID:    peerID,
				MessageID: messageID,
				Messages:  make([]bson.Raw, 0),
			}
		}

		messageGroups[groupKey].Messages = append(messageGroups[groupKey].Messages, msgRaw)
	}

	if err := cursor.Err(); err != nil {
		log.Fatal().Err(err).Msg("failed cursor")
	}

	log.Info().Int("groups", len(messageGroups)).Msg("Message grouping complete, starting migration...")

	// Мигрируем сгруппированные сообщения
	session, err := client.StartSession()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start mongo session")
	}
	defer session.EndSession(ctx)

	processedCount := 0
	for groupKey, group := range messageGroups {
		callback := func(sessCtx mongo.SessionContext) (any, error) {
			// Обновляем или создаем чат
			var chat Chat
			err := chatsCollection.FindOneAndUpdate(
				sessCtx,
				bson.M{
					"user_id": group.UserID,
					"peer_id": group.PeerID,
				},
				bson.M{
					"$setOnInsert": bson.M{
						"last_message_unixtime": -1,
						"last_message_id":       -1,
					},
				},
				options.FindOneAndUpdate().
					SetUpsert(true).
					SetReturnDocument(options.After),
			).Decode(&chat)
			if err != nil {
				return nil, fmt.Errorf("failed to upsert chat: %w", err)
			}

			group.ChatID = chat.ID

			// Создаем или обновляем документ сообщения
			_, err = messageCollection.UpdateOne(
				sessCtx,
				bson.M{
					"message_id": group.MessageID,
					"chat_ids":   chat.ID,
				},
				bson.D{
					{Key: "$addToSet", Value: bson.D{
						{Key: "chat_ids", Value: chat.ID},
					}},
					{Key: "$addToSet", Value: bson.D{
						{Key: "messages", Value: bson.M{
							"$each": group.Messages,
						}},
					}},
				},
				options.Update().SetUpsert(true),
			)
			if err != nil {
				return nil, fmt.Errorf("failed to upsert message: %w", err)
			}

			return nil, nil
		}

		_, err = session.WithTransaction(ctx, callback)
		if err != nil {
			log.Error().Err(err).Str("groupKey", groupKey).Msg("transaction failed")
			continue
		}

		processedCount++
		if processedCount%100 == 0 {
			log.Info().Int("processed", processedCount).Int("total", len(messageGroups)).Msg("Migration progress")
		}
	}

	log.Info().Msg("Starting peers update...")

	// Обновляем peer'ы
	for peerID, peerData := range peerUpdates {
		_, err = peersCollection.UpdateOne(
			ctx,
			bson.M{"_id": peerID},
			bson.M{"$set": peerData},
			options.Update().SetUpsert(true),
		)
		if err != nil {
			log.Error().Err(err).Int64("peerID", peerID).Msg("failed to update peer")
		}
	}

	log.Info().Msg("Starting last message timestamps update...")

	// Обновляем last_message_unixtime и last_message_id для всех чатов
	err = updateLastMessageTimestamps(ctx, chatsCollection, messageCollection)
	if err != nil {
		log.Error().Err(err).Msg("failed to update last message timestamps")
	}

	// Удаляем старую коллекцию
	err = chatsResolveCollection.Drop(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to drop chats resolve coll")
	}

	_, err = messageCollection.Indexes().DropAll(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to drop all indexes in messages coll")
	}

	log.Info().Msg("Migration complete")
	return nil
}

// updateLastMessageTimestamps обновляет last_message_unixtime и last_message_id для всех чатов
func updateLastMessageTimestamps(ctx context.Context, chatsCollection, messageCollection *mongo.Collection) error {
	// Получаем все чаты с невалидными временными метками
	cursor, err := chatsCollection.Find(ctx, bson.M{
		"$or": []bson.M{
			{"last_message_unixtime": -1},
			{"last_message_id": -1},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to find chats: %w", err)
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var chat Chat
		if err := cursor.Decode(&chat); err != nil {
			log.Warn().Err(err).Msg("failed to decode chat")
			continue
		}

		// Находим последнее сообщение для этого чата
		pipeline := []bson.M{
			{
				"$match": bson.M{
					"chat_ids": chat.ID,
				},
			},
			{
				"$unwind": "$messages",
			},
			{
				"$sort": bson.M{
					"messages.created_at": -1,
				},
			},
			{
				"$limit": 1,
			},
			{
				"$project": bson.M{
					"message":    "$messages.message",
					"created_at": "$messages.created_at",
				},
			},
		}

		aggCursor, err := messageCollection.Aggregate(ctx, pipeline)
		if err != nil {
			log.Warn().Err(err).Str("chatID", chat.ID.Hex()).Msg("failed to aggregate messages")
			continue
		}

		var lastMessage struct {
			Message   bson.Raw  `bson:"message"`
			CreatedAt time.Time `bson:"created_at"`
		}

		if aggCursor.Next(ctx) {
			if err := aggCursor.Decode(&lastMessage); err != nil {
				log.Warn().Err(err).Str("chatID", chat.ID.Hex()).Msg("failed to decode last message")
				aggCursor.Close(ctx)
				continue
			}

			// Парсим сообщение чтобы получить message_id
			var telegoMsg telego.Message
			if err := bson.Unmarshal(lastMessage.Message, &telegoMsg); err != nil {
				log.Warn().Err(err).Str("chatID", chat.ID.Hex()).Msg("failed to unmarshal telego message")
				aggCursor.Close(ctx)
				continue
			}

			// Обновляем чат с корректными значениями
			_, err = chatsCollection.UpdateOne(ctx,
				bson.M{"_id": chat.ID},
				bson.M{
					"$set": bson.M{
						"last_message_unixtime": lastMessage.CreatedAt.Unix(),
						"last_message_id":       telegoMsg.MessageID,
					},
				},
			)
			if err != nil {
				log.Warn().Err(err).Str("chatID", chat.ID.Hex()).Msg("failed to update chat timestamps")
			}
		} else {
			// Если сообщений нет, ставим 0
			_, err = chatsCollection.UpdateOne(ctx,
				bson.M{"_id": chat.ID},
				bson.M{
					"$set": bson.M{
						"last_message_unixtime": 0,
						"last_message_id":       0,
					},
				},
			)
			if err != nil {
				log.Warn().Err(err).Str("chatID", chat.ID.Hex()).Msg("failed to update chat with zero timestamps")
			}
		}

		aggCursor.Close(ctx)
	}

	return cursor.Err()
}
