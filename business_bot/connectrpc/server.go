package connectrpc

import (
	"fmt"
	"net/http"
	"ssuspy-bot/repository"
	"ssuspy-bot/telegram/manager"
	"ssuspy-proto/gen/manager/v1/managerv1connect"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func RunConnectRPC(
	manager *manager.BotManager,
	repo *repository.Repository,
	port int,
) error {
	mux := http.NewServeMux()

	managerService := NewManagerService(manager, repo)

	path, handler := managerv1connect.NewManagerServiceHandler(managerService)
	mux.Handle(path, handler)
	return http.ListenAndServe(
		fmt.Sprintf(":%d", port),
		h2c.NewHandler(mux, &http2.Server{}),
	)
}
