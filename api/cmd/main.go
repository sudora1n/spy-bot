package main

import (
	"context"
	"errors"
	"net/http"
	"ssuspy-api/config"
	"ssuspy-api/repository"

	"connectrpc.com/connect"
	"github.com/go-chi/chi/v5"
	"github.com/mymmrac/telego"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/protobuf/types/known/timestamppb"

	botsv1 "ssuspy-proto/gen/bots/v1"
	"ssuspy-proto/gen/bots/v1/botsv1connect"
	typesv1 "ssuspy-proto/gen/types/v1"
)

var (
	INTERNAL_ERROR     = errors.New("internal error")
	TOO_MANY_BOTS      = errors.New("too many bots")
	ALREADY_EXISTS     = errors.New("bot already exists")
	INVALID_TOKEN      = errors.New("invalid bot token")
	INCORRECT_SETTINGS = errors.New("incorrect settings in @botfather")
	PERMISSION_DENIED  = errors.New("it's not your bot ^^(")
)

func BotToProtoBot(bot *repository.Bot) *typesv1.Bot {
	return &typesv1.Bot{
		Id:        bot.ID,
		Username:  bot.Username,
		UserId:    bot.UserID,
		CreatedAt: timestamppb.New(bot.CreatedAt),
	}
}

type BotsServer struct {
	mongo repository.MongoRepository
}

func (s *BotsServer) GetBots(
	ctx context.Context,
	req *connect.Request[botsv1.GetBotsRequest],
) (*connect.Response[botsv1.GetBotsResponse], error) {
	bots, err := s.mongo.FindBots(ctx, req.Msg.UserId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, INTERNAL_ERROR)
	}

	res := connect.NewResponse(&botsv1.GetBotsResponse{})
	res.Header().Set("GetBots-Version", "v1")

	for _, bot := range bots {
		res.Msg.Bots = append(
			res.Msg.Bots,
			BotToProtoBot(&bot),
		)
	}

	return res, nil
}

func (s *BotsServer) GetBotStat(
	ctx context.Context,
	req *connect.Request[botsv1.GetBotStatRequest],
) (*connect.Response[botsv1.GetBotStatResponse], error) {
	stat, err := s.mongo.FindBotWithUserCounts(ctx, req.Msg.UserId, req.Msg.BotId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, INTERNAL_ERROR)
	}

	res := connect.NewResponse(&botsv1.GetBotStatResponse{
		BotStat: &botsv1.BotStat{
			TotalUsers:         stat.TotalUsers,
			TotalBusinessUsers: stat.TotalBusinessUsers,
			Bot:                BotToProtoBot(&stat.Bot),
		},
	})
	res.Header().Set("GetBotStat-Version", "v1")
	return res, nil
}

func (s *BotsServer) CreateBot(
	ctx context.Context,
	req *connect.Request[botsv1.CreateBotRequest],
) (*connect.Response[botsv1.CreateBotResponse], error) {
	botsLen, err := s.mongo.LenBots(ctx, req.Msg.UserId)
	if err != nil {
		log.Warn().Err(err).Int64("userID", req.Msg.UserId).Msg("failed get len of bots")
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, INTERNAL_ERROR)
		}
	}

	if botsLen > config.Config.MaxBotsByUser {
		return nil, connect.NewError(connect.CodeResourceExhausted, TOO_MANY_BOTS)
	}

	botExists, err := s.mongo.FindBotByToken(ctx, req.Msg.UserId, req.Msg.Token)
	if err != nil && err != mongo.ErrNoDocuments {
		return nil, connect.NewError(connect.CodeInternal, INTERNAL_ERROR)
	}
	if botExists != nil {
		return nil, connect.NewError(connect.CodeAlreadyExists, ALREADY_EXISTS)
	}

	newBot, err := telego.NewBot(req.Msg.Token, telego.WithAPIServer(config.Config.TelegramBot.ApiURL))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, INVALID_TOKEN)
	}

	botUser, err := newBot.GetMe(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, INTERNAL_ERROR)
	}
	if !botUser.CanConnectToBusiness || !botUser.SupportsInlineQueries {
		badRequestInfo := &errdetails.BadRequest{
			FieldViolations: []*errdetails.BadRequest_FieldViolation{},
		}

		if !botUser.CanConnectToBusiness {
			badRequestInfo.FieldViolations = append(badRequestInfo.FieldViolations, &errdetails.BadRequest_FieldViolation{
				Field: "noBusiness",
			})
		}
		if !botUser.SupportsInlineQueries {
			badRequestInfo.FieldViolations = append(badRequestInfo.FieldViolations, &errdetails.BadRequest_FieldViolation{
				Field: "noInline",
			})
		}

		err := connect.NewError(
			connect.CodeFailedPrecondition,
			INCORRECT_SETTINGS,
		)
		if detail, detailErr := connect.NewErrorDetail(badRequestInfo); detailErr == nil {
			err.AddDetail(detail)
		}

		return nil, err
	}

	err = s.mongo.InsertBot(ctx, botUser.ID, req.Msg.UserId, req.Msg.Token, botUser.Username)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, INTERNAL_ERROR)
	}

	_, err = h.grpcClient.CreateBot(ctx, &managerv1.CreateBotRequest{Id: botUser.ID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, INTERNAL_ERROR)
	}

	res := connect.NewResponse(&botsv1.CreateBotResponse{})
	res.Header().Set("CreateBot-Version", "v1")
	return res, nil
}

func (s *BotsServer) RemoveBot(
	ctx context.Context,
	req *connect.Request[botsv1.RemoveBotRequest],
) (*connect.Response[botsv1.RemoveBotResponse], error) {
	bot, err := s.mongo.BotByID(ctx, req.Msg.BotId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, INTERNAL_ERROR)
	}

	if bot.UserID != req.Msg.UserId {
		return nil, connect.NewError(connect.CodePermissionDenied, PERMISSION_DENIED)
	}

	_, err = s.grpcClient.RemoveBot(ctx, &managerv1.RemoveBotRequest{Id: req.Msg.BotId})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, INTERNAL_ERROR)
	}

	err = s.mongo.RemoveBot(ctx, req.Msg.UserId, req.Msg.BotId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, INTERNAL_ERROR)
	}

	res := connect.NewResponse(&botsv1.RemoveBotResponse{})
	res.Header().Set("RemoveBot-Version", "v1")
	return res, nil
}

func main() {
	r := chi.NewRouter()

	botsServer := &BotsServer{}
	path, handler := botsv1connect.NewBotsServiceHandler(botsServer)

	r.Mount(path, h2c.NewHandler(handler, &http2.Server{}))

	http.ListenAndServe(":3000", r)
}
