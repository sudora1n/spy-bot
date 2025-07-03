package utils

import (
	"context"
	"ssuspy-bot/repository"

	"github.com/rs/zerolog/log"
)

func ProcessBusinessBot(service *repository.MongoRepository, businessConnectionID string, userID int64, botID int64) (iUser *repository.IUser) {
	if businessConnectionID == "" && userID == 0 {
		log.Printf("No business connection ID or userID found in update")
		return nil
	}

	var err error
	if userID == 0 {
		iUser, err = service.FindIUserByConnectionID(context.Background(), businessConnectionID, botID)
		if err != nil {
			log.Warn().Str("businessConnectionID", businessConnectionID).Err(err).Msg("error finding bot user by connection")
			return nil
		}
	} else {
		iUser, err = service.FindIUserByID(context.Background(), userID, botID)
		if err != nil {
			log.Warn().Int64("userID", userID).Err(err).Msg("error finding bot user")
			return nil
		}
	}

	return iUser
}
