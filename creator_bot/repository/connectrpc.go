package repository

import (
	"net/http"
	"ssuspy-proto/gen/manager/v1/managerv1connect"

	"connectrpc.com/connect"
)

type ConnectRPCService struct {
	Manager managerv1connect.ManagerServiceClient
}

func NewConnectRPCService(businessUrl string) *ConnectRPCService {
	manager := managerv1connect.NewManagerServiceClient(
		http.DefaultClient,
		businessUrl,
		connect.WithGRPC(),
	)

	return &ConnectRPCService{
		Manager: manager,
	}
}
