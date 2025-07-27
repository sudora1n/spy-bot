package api

import (
	"context"
	"ssuspy-api/errors"
	"ssuspy-api/repository"
	usersv1 "ssuspy-proto/gen/users/v1"
	"time"

	"connectrpc.com/connect"
	"github.com/mymmrac/telego"
	"github.com/rs/zerolog/log"
	"github.com/vmihailenco/msgpack/v5"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func userToProto(user *repository.User) *usersv1.User {
	return &usersv1.User{
		Id:           user.ID,
		LanguageCode: user.LanguageCode,
		Settings: &usersv1.UserSettings{
			ShowMyEdits:        user.Settings.ShowMyEdits,
			ShowPartnerEdits:   user.Settings.ShowPartnerEdits,
			ShowMyDeleted:      user.Settings.ShowMyDeleted,
			ShowPartnerDeleted: user.Settings.ShowPartnerDeleted,
		},
		CreatedAt: timestamppb.New(time.Unix(user.CreatedAt, 0)),
	}
}

func botUserToProto(botUser *repository.BotUser) *usersv1.BotUser {
	businessConnections := make([]*usersv1.BotUserBusinessConnection, 0, len(botUser.BusinessConnections))
	for _, connection := range botUser.BusinessConnections {
		rights, err := msgpack.Marshal(connection.Rights)
		if err != nil {
			log.Warn().Err(err).Any("rights", connection.Rights).Msg("failed msgpack rights")
		}

		businessConnections = append(businessConnections, &usersv1.BotUserBusinessConnection{
			Id:       connection.ID,
			Rights:   rights,
			Enabled:  connection.Enabled,
			Unixtime: connection.Unixtime,
		})
	}

	return &usersv1.BotUser{
		InternalId:          botUser.InternalID,
		BusinessConnections: businessConnections,
		SendMessages:        botUser.SendMessages,
		UserId:              botUser.UserID,
		BotId:               botUser.BotID,
		CreatedAt:           botUser.CreatedAt,
	}
}

type UsersServer struct {
	mongo repository.MongoRepository
}

func (s *UsersServer) UpdateUserSendMessages(
	ctx context.Context,
	req *connect.Request[usersv1.UpdateUserSendMessagesRequest],
) (*connect.Response[usersv1.UpdateUserSendMessagesResponse], error) {
	err := s.mongo.UpdateUserSendMessages(ctx, req.Msg.UserId, req.Msg.SendMessages)
	if err != nil {
		return nil, errors.CONNECT_ERROR_INTERNAL
	}

	res := connect.NewResponse(&usersv1.UpdateUserSendMessagesResponse{})
	res.Header().Set("UpdateUserSendMessages-Version", "v1")
	return res, nil
}

func (s *UsersServer) UpdateUserLanguage(
	ctx context.Context,
	req *connect.Request[usersv1.UpdateUserLanguageRequest],
) (*connect.Response[usersv1.UpdateUserLanguageResponse], error) {
	err := s.mongo.UpdateUserLanguage(ctx, req.Msg.UserId, req.Msg.LanguageCode)
	if err != nil {
		return nil, errors.CONNECT_ERROR_INTERNAL
	}

	res := connect.NewResponse(&usersv1.UpdateUserLanguageResponse{})
	res.Header().Set("UpdateUserLanguage-Version", "v1")
	return res, nil
}

func (s *UsersServer) UpdateBotUserConnection(
	ctx context.Context,
	req *connect.Request[usersv1.UpdateBotUserConnectionRequest],
) (*connect.Response[usersv1.UpdateBotUserConnectionResponse], error) {
	var businessConnection *telego.BusinessConnection

	err := msgpack.Unmarshal(req.Msg.BusinessConnection, &businessConnection)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.ERROR_INVALID_PARAM)
	}

	isUpdated, err := s.mongo.UpdateBotUserConnection(ctx, businessConnection, req.Msg.BotId)
	if err != nil {
		return nil, errors.CONNECT_ERROR_INTERNAL
	}

	res := connect.NewResponse(&usersv1.UpdateBotUserConnectionResponse{
		IsUpdated: isUpdated,
	})
	res.Header().Set("UpdateBotUserConnection-Version", "v1")
	return res, nil
}

func (s *UsersServer) UpdateBotUserSendMessages(
	ctx context.Context,
	req *connect.Request[usersv1.UpdateBotUserSendMessagesRequest],
) (*connect.Response[usersv1.UpdateBotUserSendMessagesResponse], error) {
	err := s.mongo.UpdateBotUserSendMessages(ctx, req.Msg.UserId, req.Msg.BotId, req.Msg.SendMessages)
	if err != nil {
		return nil, errors.CONNECT_ERROR_INTERNAL
	}

	res := connect.NewResponse(&usersv1.UpdateBotUserSendMessagesResponse{})
	res.Header().Set("UpdateBotUserSendMessages-Version", "v1")
	return res, nil
}

func (s *UsersServer) GetUserWithBotUser(
	ctx context.Context,
	req *connect.Request[usersv1.GetUserWithBotUserRequest],
) (*connect.Response[usersv1.GetUserWithBotUserResponse], error) {
	var (
		iUser *repository.IUser
		err   error
	)

	switch v := req.Msg.Id.(type) {
	case *usersv1.GetUserWithBotUserRequest_UserId:
		iUser, err = s.mongo.FindIUserByID(ctx, v.UserId, req.Msg.BotId)
	case *usersv1.GetUserWithBotUserRequest_BusinessConnectionId:
		iUser, err = s.mongo.FindIUserByConnectionID(ctx, v.BusinessConnectionId, req.Msg.BotId)
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.ERROR_INVALID_PARAM)
	}

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.CONNECT_USER_NOT_FOUND
		}
		return nil, errors.CONNECT_ERROR_INTERNAL
	}

	res := connect.NewResponse(&usersv1.GetUserWithBotUserResponse{
		User:    userToProto(&iUser.User),
		BotUser: botUserToProto(iUser.BotUser),
	})
	res.Header().Set("GetUserWithBotUser-Version", "v1")
	return res, nil
}

func (s *UsersServer) CreateUser(
	ctx context.Context,
	req *connect.Request[usersv1.CreateUserRequest],
) (*connect.Response[usersv1.CreateUserResponse], error) {
	languageCode := req.Msg.GetLanguageCode()
	if languageCode == "" {
		languageCode = "en"
	}

	err := s.mongo.CreateUser(
		ctx,
		req.Msg.UserId,
		languageCode,
		req.Msg.GetFromCreator(),
	)
	if err != nil {
		return nil, errors.CONNECT_ERROR_INTERNAL
	}

	res := connect.NewResponse(&usersv1.CreateUserResponse{})
	res.Header().Set("CreateUser-Version", "v1")
	return res, nil
}

func (s *UsersServer) GetUser(
	ctx context.Context,
	req *connect.Request[usersv1.GetUserRequest],
) (*connect.Response[usersv1.GetUserResponse], error) {
	user, err := s.mongo.FindUser(
		ctx,
		req.Msg.UserId,
	)
	if err != nil {
		return nil, errors.CONNECT_ERROR_INTERNAL
	}

	res := connect.NewResponse(&usersv1.GetUserResponse{
		User: userToProto(user),
	})
	res.Header().Set("GetUser-Version", "v1")
	return res, nil
}

func (s *UsersServer) CreateBotUser(
	ctx context.Context,
	req *connect.Request[usersv1.CreateBotUserRequest],
) (*connect.Response[usersv1.CreateBotUserResponse], error) {
	err := s.mongo.CreateBotUser(ctx, req.Msg.UserId, req.Msg.BotId)
	if err != nil {
		return nil, errors.CONNECT_ERROR_INTERNAL
	}

	res := connect.NewResponse(&usersv1.CreateBotUserResponse{})
	res.Header().Set("CreateBotUser-Version", "v1")
	return res, nil
}

func (s *UsersServer) GetBotUser(
	ctx context.Context,
	req *connect.Request[usersv1.GetBotUserRequest],
) (*connect.Response[usersv1.GetBotUserResponse], error) {
	botUser, err := s.mongo.FindBotUser(
		ctx,
		req.Msg.UserId,
		req.Msg.BotId,
	)
	if err != nil {
		return nil, errors.CONNECT_ERROR_INTERNAL
	}

	res := connect.NewResponse(&usersv1.GetBotUserResponse{
		BotUser: botUserToProto(botUser),
	})
	res.Header().Set("GetBotUser-Version", "v1")
	return res, nil
}
