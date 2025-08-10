package handlers

import (
	"ssuspy-bot/repository"
)

type Handler struct {
	repository *repository.Repository
}

func NewHandlerGroup(repository *repository.Repository) *Handler {
	return &Handler{
		repository: repository,
	}
}
