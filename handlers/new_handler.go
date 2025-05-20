package handlers

import (
	"ssuspy-bot/redis"
	"ssuspy-bot/repository"
)

type Handler struct {
	service *repository.MongoRepository
	rdb     *redis.Redis
}

func NewHandlerGroup(service *repository.MongoRepository, rdb *redis.Redis) *Handler {
	return &Handler{
		service: service,
		rdb:     rdb,
	}
}
