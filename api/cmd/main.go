package main

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	bots "ssuspy-proto/gen/bots/v1"
	"ssuspy-proto/gen/bots/v1/botsv1connect"
)

type BotsServer struct{}

func (s *BotsServer) GetBots(
	ctx context.Context,
	req *connect.Request[bots.GetBotsRequest],
) (*connect.Response[bots.GetBotsResponse], error) {
	res := connect.NewResponse(&bots.GetBotsResponse{})
	res.Header().Set("GetBots-Version", "v1")
	return res, nil
}

func (s *BotsServer) GetBotStat(
	ctx context.Context,
	req *connect.Request[bots.GetBotStatRequest],
) (*connect.Response[bots.GetBotStatResponse], error) {
	res := connect.NewResponse(&bots.GetBotStatResponse{})
	res.Header().Set("GetBotStat-Version", "v1")
	return res, nil
}

func (s *BotsServer) GetBotByTokenHash(
	ctx context.Context,
	req *connect.Request[bots.GetBotByTokenHashRequest],
) (*connect.Response[bots.GetBotByTokenHashResponse], error) {
	res := connect.NewResponse(&bots.GetBotByTokenHashResponse{})
	res.Header().Set("GetBotByTokenHash-Version", "v1")
	return res, nil
}

func (s *BotsServer) CreateBot(
	ctx context.Context,
	req *connect.Request[bots.CreateBotRequest],
) (*connect.Response[bots.CreateBotResponse], error) {
	res := connect.NewResponse(&bots.CreateBotResponse{})
	res.Header().Set("CreateBot-Version", "v1")
	return res, nil
}

func (s *BotsServer) RemoveBot(
	ctx context.Context,
	req *connect.Request[bots.RemoveBotRequest],
) (*connect.Response[bots.RemoveBotResponse], error) {
	res := connect.NewResponse(&bots.RemoveBotResponse{})
	res.Header().Set("RemoveBot-Version", "v1")
	return res, nil
}

func main() {
	botsServer := &BotsServer{}
	mux := http.NewServeMux()
	path, handler := botsv1connect.NewBotsServiceHandler(botsServer)
	mux.Handle(path, handler)
	http.ListenAndServe(
		"localhost:8080",
		h2c.NewHandler(mux, &http2.Server{}),
	)
}
