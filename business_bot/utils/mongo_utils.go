package utils

import (
	"context"
	"ssuspy-bot/repository"

	"github.com/rs/zerolog/log"
)

func ProcessBusiness(service *repository.MongoRepository, businessConnectionID string, userID int64) (user *repository.User) {
	if businessConnectionID == "" && userID == 0 {
		log.Printf("No business connection ID or userID found in update")
		return nil
	}

	var err error
	if userID == 0 {
		user, err = service.FindUserByConnectionID(context.Background(), businessConnectionID)
		if err != nil {
			log.Warn().Err(err).Msg("error decoding user")
			return nil
		}
	} else {
		user, err = service.FindUser(context.Background(), userID)
		if err != nil {
			log.Warn().Err(err).Msg("error decoding user")
			return nil
		}
	}

	return user
}
