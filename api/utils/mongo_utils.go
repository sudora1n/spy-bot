package utils

import (
	"go.mongodb.org/mongo-driver/mongo"
)

func ErrIsAlreadyExists(err error) bool {
	if mongoErr, ok := err.(mongo.WriteException); ok {
		for _, we := range mongoErr.WriteErrors {
			if we.Code == 11000 {
				return true
			}
		}
	}

	return false
}
