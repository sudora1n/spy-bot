package connectrpc

import (
	"context"
	"fmt"
	manager_proto "ssuspy-proto/gen/manager/v1"
	managerv1 "ssuspy-proto/gen/manager/v1"

	"connectrpc.com/connect"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"ssuspy-bot/repository"
	"ssuspy-bot/telegram/manager"
)

type ManagerService struct {
	manager    *manager.BotManager
	repository *repository.Repository
}

func NewManagerService(manager *manager.BotManager, repo *repository.Repository) *ManagerService {
	return &ManagerService{
		manager:    manager,
		repository: repo,
	}
}
func (s *ManagerService) CreateBot(
	ctx context.Context,
	req *connect.Request[managerv1.CreateBotRequest],
) (*connect.Response[managerv1.CreateBotResponse], error) {
	if req.Msg.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "botID is required")
	}

	bot, err := s.repository.Mongo.BotByID(ctx, req.Msg.Id)
	if err != nil {
		log.Error().Err(err).Int64("botID", req.Msg.Id).Msg("failed to get bot from database")
		return nil, status.Error(codes.NotFound, "bot not found in database")
	}

	err = s.manager.AddBot(ctx, req.Msg.Id, bot.SecretToken)
	if err != nil {
		log.Error().Err(err).Int64("botID", req.Msg.Id).Msg("failed to add bot")
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to add bot: %v", err))
	}

	botInstance, exists := s.manager.GetBot(req.Msg.Id)
	if !exists {
		return nil, status.Error(codes.Internal, "bot was added but not found in manager")
	}

	botInfo, err := botInstance.Bot.GetMe(ctx)
	if err != nil {
		log.Error().Err(err).Int64("botID", req.Msg.Id).Msg("failed to get bot info")
		return connect.NewResponse(&manager_proto.CreateBotResponse{
			Id:       bot.Id,
			Username: "unknown",
		}), nil
	}

	log.Info().Int64("botID", req.Msg.Id).Str("username", botInfo.Username).Msg("bot added successfully")

	res := connect.NewResponse(&managerv1.CreateBotResponse{
		Id:       req.Msg.Id,
		Username: botInfo.Username,
	})
	res.Header().Set("CreateBot-Version", "v1")
	return res, nil
}

func (s *ManagerService) RemoveBot(
	ctx context.Context,
	req *connect.Request[managerv1.RemoveBotRequest],
) (*connect.Response[managerv1.RemoveBotResponse], error) {
	if req.Msg.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "botID is required")
	}

	botInstance, exists := s.manager.GetBot(req.Msg.Id)
	var username string = "unknown"

	if exists {
		botInfo, err := botInstance.Bot.GetMe(ctx)
		if err == nil {
			username = botInfo.Username
		}
	}

	err := s.manager.RemoveBot(req.Msg.Id)
	if err != nil {
		log.Error().Err(err).Int64("botID", req.Msg.Id).Msg("failed to remove bot")
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to remove bot: %v", err))
	}

	log.Info().Int64("botID", req.Msg.Id).Str("username", username).Msg("bot removed successfully")

	res := connect.NewResponse(&managerv1.RemoveBotResponse{
		Id:       req.Msg.Id,
		Username: username,
	})
	res.Header().Set("RemoveBot-Version", "v1")
	return res, nil
}
