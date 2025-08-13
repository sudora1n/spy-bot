package createBot

import (
	"context"
	"errors"
	"fmt"
	"ssuspy-creator-bot/config"
	"ssuspy-creator-bot/repository"
	managerv1 "ssuspy-proto/gen/manager/v1"

	"connectrpc.com/connect"
	"github.com/mymmrac/telego"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
)

type CreateBotError struct {
	Code            CreateBotStatus
	InvalidSettings []string
}

func (e *CreateBotError) Error() string {
	msg := fmt.Sprintf("create bot error: code %d", e.Code)
	if len(e.InvalidSettings) > 0 {
		msg += fmt.Sprintf(", invalid settings: %v", e.InvalidSettings)
	}
	return msg
}

func NewCreateBotError(code CreateBotStatus) *CreateBotError {
	return &CreateBotError{Code: code}
}

func NewCreateBotErrorWithInvalidSettings(code CreateBotStatus, invalidSettings []string) *CreateBotError {
	return &CreateBotError{
		Code:            code,
		InvalidSettings: invalidSettings,
	}
}

type CreateBotStatus uint8

const (
	STATUS_CREATEBOT_OK CreateBotStatus = iota
	STATUS_CREATEBOT_ERROR_INTERNAL
	STATUS_CREATEBOT_ERROR_TOO_MANY_BOTS
	STATUS_CREATEBOT_ERROR_BOT_ALREADY_EXISTS
	STATUS_CREATEBOT_ERROR_BOT_INVALID
	STATUS_CREATEBOT_ERROR_BOT_INVALID_SETTINGS
)

type CreateBotOkInfo struct {
	Username string
}

type CreateBotInvalidSettingsErrorInfo struct {
	Fields []string
}

type CreateBotInfo interface {
	isCreateBotInfo()
}

func (CreateBotOkInfo) isCreateBotInfo()                   {}
func (CreateBotInvalidSettingsErrorInfo) isCreateBotInfo() {}

func CreateBot(
	ctx context.Context,
	repository *repository.Repository,
	userID int64,
	botToken string,
) (
	string, // username of bot, if success
	error,
) {
	botsLen, err := repository.Mongo.LenBots(ctx, userID)
	if err != nil {
		log.Warn().Err(err).Int64("userID", userID).Msg("failed get len of bots")
		return "", NewCreateBotError(STATUS_CREATEBOT_ERROR_INTERNAL)
	}

	if botsLen > config.Config.MaxBotsByUser {
		return "", NewCreateBotError(STATUS_CREATEBOT_ERROR_TOO_MANY_BOTS)
	}

	botExists, err := repository.Mongo.FindBotByToken(ctx, userID, botToken)
	if err != nil && errors.Is(err, mongo.ErrNoDocuments) {
		return "", NewCreateBotError(STATUS_CREATEBOT_ERROR_INTERNAL)
	}
	if botExists != nil {
		return "", NewCreateBotError(STATUS_CREATEBOT_ERROR_BOT_ALREADY_EXISTS)
	}

	newBot, err := telego.NewBot(botToken, telego.WithAPIServer(config.Config.TelegramBot.ApiURL))
	if err != nil {
		return "", NewCreateBotError(STATUS_CREATEBOT_ERROR_BOT_INVALID)
	}

	botUser, err := newBot.GetMe(ctx)
	if err != nil {
		return "", NewCreateBotError(STATUS_CREATEBOT_ERROR_INTERNAL)
	}
	if !botUser.CanConnectToBusiness || !botUser.SupportsInlineQueries {
		errFields := make([]string, 0, 2)

		if !botUser.CanConnectToBusiness {
			errFields = append(errFields, "noBusiness")
		}
		if !botUser.SupportsInlineQueries {
			errFields = append(errFields, "noInline")
		}

		return "", NewCreateBotErrorWithInvalidSettings(STATUS_CREATEBOT_ERROR_BOT_INVALID_SETTINGS, errFields)
	}

	err = repository.Mongo.InsertBot(ctx, botUser.ID, userID, botToken, botUser.Username)
	if err != nil {
		return "", NewCreateBotError(STATUS_CREATEBOT_ERROR_INTERNAL)
	}

	_, err = repository.ConnectRPC.Manager.CreateBot(ctx, connect.NewRequest(&managerv1.CreateBotRequest{Id: botUser.ID}))
	if err != nil {
		return "", NewCreateBotError(STATUS_CREATEBOT_ERROR_INTERNAL)
	}

	return botUser.Username, nil
}
