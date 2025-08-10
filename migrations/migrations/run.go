package migrations

import (
	"context"
	custom_registry "ssuspy-common/bson"
	"ssuspy-migrations/repository"
)

func RunMigrations(repo *repository.Repository) error {
	ctx := context.Background()

	db := repo.Mongo.GetDb()
	customRegistry := custom_registry.CreateCustomRegistry()

	jsonToBsonMessages := "json_to_bson_messages"
	JsonToBsonMessagesMigrationIsNeeded, err := repo.Mongo.MigrationIsNeeded(ctx, jsonToBsonMessages)
	if err != nil {
		return err
	}
	if JsonToBsonMessagesMigrationIsNeeded {
		DoJsonToBsonMessagesMigrate(context.Background(), db, customRegistry)
		if err := repo.Mongo.ApplyMigration(ctx, jsonToBsonMessages); err != nil {
			return err
		}
	}

	addUserSettings := "add_user_settings"
	AddUserSettingsIsNeeded, err := repo.Mongo.MigrationIsNeeded(ctx, addUserSettings)
	if err != nil {
		return err
	}
	if AddUserSettingsIsNeeded {
		DoAddUserSettingsMigrate(context.Background(), db.Collection("users"))
		if err := repo.Mongo.ApplyMigration(ctx, addUserSettings); err != nil {
			return err
		}
	}

	userSettingsShorter := "user_settings_shorter"
	UserSettingsShorterIsNeeded, err := repo.Mongo.MigrationIsNeeded(ctx, userSettingsShorter)
	if err != nil {
		return err
	}
	if UserSettingsShorterIsNeeded {
		DoUserSettingsShorterMigrate(context.Background(), db.Collection("users"))
		if err := repo.Mongo.ApplyMigration(ctx, userSettingsShorter); err != nil {
			return err
		}
	}

	autoincrementToObjectID := "counter_to_objectid"
	autoincrementToObjectIDIsNeeded, err := repo.Mongo.MigrationIsNeeded(ctx, autoincrementToObjectID)
	if err != nil {
		return err
	}
	if autoincrementToObjectIDIsNeeded {
		err = db.Collection("counters").Drop(ctx)
		if err != nil {
			return err
		}

		collsList := []string{
			"files_exists",
			"bot_users",
			"telegram_messages_v2",
			"callback_data_deleted",
			"callback_data_edited",
		}

		for _, coll := range collsList {
			err = MigrateCollectionToObjectID(ctx, db, coll)
			if err != nil {
				return err
			}
		}
		if err := repo.Mongo.ApplyMigration(ctx, autoincrementToObjectID); err != nil {
			return err
		}
	}

	messagesV3 := "messagesv3"
	messagesV3IsNeeded, err := repo.Mongo.MigrationIsNeeded(ctx, messagesV3)
	if err != nil {
		return err
	}
	if messagesV3IsNeeded {
		DoMigrateToTelegramMessagesV3(
			ctx,
			db.Client(),
			customRegistry,
			db.Collection("telegram_messages_v3"),
			db.Collection("telegram_messages_v2"),
			db.Collection("bot_users"),
			db.Collection("peers"),
			db.Collection("chats"),
			db.Collection("chats_resolve"),
		)
		if err := repo.Mongo.ApplyMigration(ctx, messagesV3); err != nil {
			return err
		}
	}

	return nil
}
