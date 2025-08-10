package handlers

import (
	"context"
	"errors"
	"net/http"
	"ssuspy-api/repository"
	"ssuspy-api/service/jwt"
	"ssuspy-api/service/user"
	authv1 "ssuspy-proto/gen/auth/v1"
	"strconv"
	"time"

	"connectrpc.com/connect"
	initdataChecker "github.com/telegram-mini-apps/init-data-golang"
)

var (
	ErrAuthInitDataInvalidArgument = errors.New("initdata is not valid")
	ErrAuthBotInvalidArgument      = errors.New("bot not found")
	ErrAuthIssueToken              = errors.New("issue jwt error")
	ErrAuthSyncUser                = errors.New("error while syncing user")

	ConnectErrAuthInitDataInvalidArgument = connect.NewError(connect.CodeInvalidArgument, ErrAuthInitDataInvalidArgument)
	ConnectErrAuthBotInvalidArgument      = connect.NewError(connect.CodeInvalidArgument, ErrAuthBotInvalidArgument)
	ConnectErrAuthIssueToken              = connect.NewError(connect.CodeInternal, ErrAuthIssueToken)
	ConnectErrAuthSyncUser                = connect.NewError(connect.CodeInternal, ErrAuthSyncUser)
)

type AuthServer struct {
	repo        *repository.Repository
	jwtService  *jwt.JwtService
	userService *user.UserService
}

func NewAuthServer(repo *repository.Repository, jwtService *jwt.JwtService, userService *user.UserService) *AuthServer {
	return &AuthServer{
		repo:        repo,
		jwtService:  jwtService,
		userService: userService,
	}
}

func (a *AuthServer) AuthViaTelegramInitData(
	ctx context.Context,
	req *connect.Request[authv1.AuthViaTelegramInitDataRequest],
) (*connect.Response[authv1.AuthViaTelegramInitDataResponse], error) {
	bot, err := a.repo.Mongo.BotByID(ctx, req.Msg.BotId)
	if err != nil {
		return nil, ConnectErrAuthBotInvalidArgument
	}

	err = initdataChecker.Validate(req.Msg.RawInitData, bot.SecretToken, 24*time.Hour)
	if err != nil {
		return nil, ConnectErrAuthInitDataInvalidArgument
	}

	initdata, err := initdataChecker.Parse(req.Msg.RawInitData)
	if err != nil {
		return nil, ConnectErrAuthInitDataInvalidArgument
	}

	_, err = a.userService.SyncUser(ctx, initdata.User.ID, initdata.User.LanguageCode)
	if err != nil {
		return nil, ConnectErrAuthSyncUser
	}

	userIdStr := strconv.FormatInt(initdata.User.ID, 10)

	token, err := a.jwtService.IssueJwt(userIdStr)
	if err != nil {
		return nil, ConnectErrAuthIssueToken
	}

	cookie := &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(24 * time.Hour),
	}

	res := connect.NewResponse(&authv1.AuthViaTelegramInitDataResponse{})
	res.Header().Set("AuthViaTelegramInitData-Version", "v1")
	res.Header().Add("Set-Cookie", cookie.String())

	return res, nil
}
