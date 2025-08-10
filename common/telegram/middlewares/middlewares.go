package middleware

import (
	"errors"
	"ssuspy-bot/repository/redis"
	"ssuspy-common/repository/mongoRepository"
	"ssuspy-common/types"
	"ssuspy-proto/gen/bots/v1/botsv1connect"
	usersv1 "ssuspy-proto/gen/users/v1"
	"ssuspy-proto/gen/users/v1/usersv1connect"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/rs/zerolog/log"
)

type USER = usersv1.User

type MiddlewareGroup struct {
	rdb   *redis.Redis
	bots  botsv1connect.BotsServiceClient
	users usersv1connect.UsersServiceClient
}

func NewMiddlewareGroup(rdb *redis.Redis, bots botsv1connect.BotsServiceClient, users usersv1connect.UsersServiceClient) *MiddlewareGroup {
	return &MiddlewareGroup{
		bots:  bots,
		users: users,
		rdb:   rdb,
	}
}

func (h *MiddlewareGroup) GetInternalUserMiddleware(c *th.Context, update telego.Update) error {
	var (
		user         *mongoRepository.User
		internalUser types.InternalUser
	)

	switch {
	case update.Message != nil:
		internalUser = types.InternalUser{
			ID:           update.Message.From.ID,
			FirstName:    update.Message.From.FirstName,
			LastName:     update.Message.From.LastName,
			LanguageCode: update.Message.From.LanguageCode,
			SendMessages: true,
		}
	case update.CallbackQuery != nil:
		internalUser = types.InternalUser{
			ID:           update.CallbackQuery.From.ID,
			FirstName:    update.CallbackQuery.From.FirstName,
			LastName:     update.CallbackQuery.From.LastName,
			LanguageCode: update.CallbackQuery.From.LanguageCode,
		}
	case update.MyChatMember != nil:
		internalUser = types.InternalUser{
			ID:           update.MyChatMember.From.ID,
			FirstName:    update.MyChatMember.From.FirstName,
			LastName:     update.MyChatMember.From.LastName,
			LanguageCode: update.MyChatMember.From.LanguageCode,
			SendMessages: false,
		}
	default:
		return errors.New("userID not found")
	}

	c = c.WithValue("user", user)
	c = c.WithValue("internalUser", &internalUser)

	logger := log.With().Int64("userID", internalUser.ID).Logger()
	c = c.WithValue("log", &logger)

	return c.Next(update)
}

func AutoRespond(c *th.Context, update telego.Update) error {
	if update.CallbackQuery != nil {
		defer func() {
			c.Bot().AnswerCallbackQuery(c, tu.CallbackQuery(update.CallbackQuery.ID))
		}()
	}

	return c.Next(update)
}
