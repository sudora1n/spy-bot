package handlers

import (
	"ssuspy-bot/repository"
)

type Handler struct {
	repo *repository.Repository
}

func NewHandlerGroup(repository *repository.Repository) *Handler {
	return &Handler{
		repo: repository,
	}
}
