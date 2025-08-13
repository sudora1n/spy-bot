package user

import (
	"context"
	"errors"
	"fmt"
	"ssuspy-api/repository"
	"ssuspy-common/repository/mongoRepository"

	"go.mongodb.org/mongo-driver/mongo"
)

type UserService struct {
	repo *repository.Repository
}

func NewUserService(repo *repository.Repository) *UserService {
	return &UserService{
		repo: repo,
	}
}

func (u *UserService) SyncUser(
	ctx context.Context,
	userId int64,
	langCode string,
) (user *mongoRepository.User, err error) {
	user, err = u.repo.Mongo.FindUser(ctx, userId)
	if err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("failed to get user: %w", err)
		}

		if langCode == "" {
			langCode = "en"
		}
		err = u.repo.Mongo.CreateUser(ctx, userId, langCode, false)
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}

		user, err = u.repo.Mongo.FindUser(ctx, userId)
		if err != nil {
			return nil, fmt.Errorf("failed to get user after create: %w", err)
		}
	}

	return user, nil
}
