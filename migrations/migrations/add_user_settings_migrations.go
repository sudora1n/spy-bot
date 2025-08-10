package migrations

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func DoAddUserSettingsMigrate(ctx context.Context, userCollection *mongo.Collection) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	if _, err := userCollection.UpdateMany(
		ctx,
		bson.M{"settings": bson.M{"$exists": false}},
		bson.M{
			"$set": bson.M{
				"settings": bson.M{
					"edited": bson.M{
						"show_my_edits":      false,
						"show_partner_edits": true,
					},
					"deleted": bson.M{
						"show_my_deleted":      true,
						"show_partner_deleted": true,
					},
				},
			},
		},
	); err != nil {
		return err
	}

	log.Info().Msg("migration done")
	return nil
}
