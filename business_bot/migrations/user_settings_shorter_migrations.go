package migrations

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type OldSettings struct {
	Edited  *EditedSettings  `bson:"edited,omitempty"`
	Deleted *DeletedSettings `bson:"deleted,omitempty"`
}

type EditedSettings struct {
	ShowMyEdits      *bool `bson:"show_my_edits,omitempty"`
	ShowPartnerEdits *bool `bson:"show_partner_edits,omitempty"`
}

type DeletedSettings struct {
	ShowMyDeleted      *bool `bson:"show_my_deleted,omitempty"`
	ShowPartnerDeleted *bool `bson:"show_partner_deleted,omitempty"`
}

type NewSettings struct {
	ShowMyEdits        bool `bson:"show_my_edits"`
	ShowPartnerEdits   bool `bson:"show_partner_edits"`
	ShowMyDeleted      bool `bson:"show_my_deleted"`
	ShowPartnerDeleted bool `bson:"show_partner_deleted"`
}

func DoUserSettingsShorterMigrate(ctx context.Context, userCollection *mongo.Collection) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	cursor, err := userCollection.Find(ctx, bson.M{"settings": bson.M{"$exists": true}})
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var user struct {
			ID       int64        `bson:"_id"`
			Settings *OldSettings `bson:"settings,omitempty"`
		}

		if err := cursor.Decode(&user); err != nil {
			log.Printf("decode error: %v", err)
			continue
		}

		newSettings := NewSettings{
			ShowMyEdits:        false,
			ShowPartnerEdits:   true,
			ShowMyDeleted:      true,
			ShowPartnerDeleted: true,
		}

		if user.Settings != nil {
			if user.Settings.Edited != nil {
				if user.Settings.Edited.ShowMyEdits != nil {
					newSettings.ShowMyEdits = *user.Settings.Edited.ShowMyEdits
				}
				if user.Settings.Edited.ShowPartnerEdits != nil {
					newSettings.ShowPartnerEdits = *user.Settings.Edited.ShowPartnerEdits
				}
			}
			if user.Settings.Deleted != nil {
				if user.Settings.Deleted.ShowMyDeleted != nil {
					newSettings.ShowMyDeleted = *user.Settings.Deleted.ShowMyDeleted
				}
				if user.Settings.Deleted.ShowPartnerDeleted != nil {
					newSettings.ShowPartnerDeleted = *user.Settings.Deleted.ShowPartnerDeleted
				}
			}
		}

		_, err := userCollection.UpdateByID(ctx, user.ID, bson.M{
			"$set": bson.M{
				"settings": newSettings,
			},
		})
		if err != nil {
			log.Printf("failed to update user %d: %v", user.ID, err)
		} else {
			fmt.Printf("updated user %d\n", user.ID)
		}
	}

	log.Info().Msg("migration done")
	return nil
}
