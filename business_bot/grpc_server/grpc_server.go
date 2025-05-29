package grpc_server

import (
	"context"
	"fmt"
	"net"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"ssuspy-bot/manager"
	pb "ssuspy-bot/pb"
	"ssuspy-bot/repository"
)

type BotServer struct {
	pb.UnimplementedBotServer
	manager *manager.BotManager
	repo    *repository.MongoRepository
}

func NewBotServer(manager *manager.BotManager, repo *repository.MongoRepository) *BotServer {
	return &BotServer{
		manager: manager,
		repo:    repo,
	}
}
func (s *BotServer) AddBot(ctx context.Context, req *pb.AddBotRequest) (*pb.AddBotReply, error) {
	if req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "botID is required")
	}

	botData, err := s.repo.BotByID(ctx, req.Id)
	if err != nil {
		log.Error().Err(err).Int64("botID", req.Id).Msg("failed to get bot from database")
		return nil, status.Error(codes.NotFound, "bot not found in database")
	}

	err = s.manager.AddBot(ctx, req.Id, botData.SecretToken)
	if err != nil {
		log.Error().Err(err).Int64("botID", req.Id).Msg("failed to add bot")
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to add bot: %v", err))
	}

	botInstance, exists := s.manager.GetBot(req.Id)
	if !exists {
		return nil, status.Error(codes.Internal, "bot was added but not found in manager")
	}

	botInfo, err := botInstance.Bot.GetMe(ctx)
	if err != nil {
		log.Error().Err(err).Int64("botID", req.Id).Msg("failed to get bot info")
		return &pb.AddBotReply{
			Id:       req.Id,
			Username: "unknown",
		}, nil
	}

	log.Info().Int64("botID", req.Id).Str("username", botInfo.Username).Msg("bot added successfully")

	return &pb.AddBotReply{
		Id:       req.Id,
		Username: botInfo.Username,
	}, nil
}

func (s *BotServer) RemoveBot(ctx context.Context, req *pb.RemoveBotRequest) (*pb.RemoveBotReply, error) {
	if req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "botID is required")
	}

	botInstance, exists := s.manager.GetBot(req.Id)
	var username string = "unknown"

	if exists {
		botInfo, err := botInstance.Bot.GetMe(ctx)
		if err == nil {
			username = botInfo.Username
		}
	}

	err := s.manager.RemoveBot(req.Id)
	if err != nil {
		log.Error().Err(err).Int64("botID", req.Id).Msg("failed to remove bot")
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to remove bot: %v", err))
	}

	log.Info().Int64("botID", req.Id).Str("username", username).Msg("bot removed successfully")

	return &pb.RemoveBotReply{
		Id:       req.Id,
		Username: username,
	}, nil
}

func StartGRPCServer(port string, manager *manager.BotManager, repo *repository.MongoRepository) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	botServer := NewBotServer(manager, repo)

	pb.RegisterBotServer(grpcServer, botServer)

	log.Info().Str("port", port).Msg("starting gRPC server")
	return grpcServer.Serve(lis)
}
