package handlers

import (
	"context"
	"ssuspy-api/errors"
	"ssuspy-api/repository"
	"ssuspy-api/types"
	mongo_repository "ssuspy-common/repository/mongoRepository"
	usersv1 "ssuspy-proto/gen/users/v1"
	"time"

	"connectrpc.com/authn"
	"connectrpc.com/connect"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func userToProto(user *mongo_repository.User) *usersv1.User {
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

type UsersServer struct {
	repo *repository.Repository
}

func NewUsersServer(repo *repository.Repository) *UsersServer {
	return &UsersServer{}
}

func (s *UsersServer) UpdateLanguage(
	ctx context.Context,
	req *connect.Request[usersv1.UpdateLanguageRequest],
) (*connect.Response[usersv1.UpdateLanguageResponse], error) {
	user := authn.GetInfo(ctx).(*types.User)

	err := s.repo.Mongo.UpdateUserLanguage(ctx, user.Id, req.Msg.LanguageCode)
	if err != nil {
		return nil, errors.CONNECT_ERROR_INTERNAL
	}

	res := connect.NewResponse(&usersv1.UpdateLanguageResponse{})
	res.Header().Set("UpdateLanguage-Version", "v1")
	return res, nil
}

func (s *UsersServer) GetMe(
	ctx context.Context,
	req *connect.Request[usersv1.GetMeRequest],
) (*connect.Response[usersv1.GetMeResponse], error) {
	user := authn.GetInfo(ctx).(*types.User)

	userRes, err := s.repo.Mongo.FindUser(
		ctx,
		user.Id,
	)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.CONNECT_USER_NOT_FOUND
		}
		return nil, errors.CONNECT_ERROR_INTERNAL
	}

	res := connect.NewResponse(&usersv1.GetMeResponse{
		User: userToProto(userRes),
	})
	res.Header().Set("GetMe-Version", "v1")
	return res, nil
}
