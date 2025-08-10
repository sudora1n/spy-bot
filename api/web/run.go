package web

import (
	"fmt"
	"net/http"
	"ssuspy-api/config"
	"ssuspy-api/repository"
	"ssuspy-api/service/jwt"
	"ssuspy-api/service/user"
	"ssuspy-api/web/handlers"
	"ssuspy-api/web/middlewares"
	"ssuspy-proto/gen/auth/v1/authv1connect"
	"ssuspy-proto/gen/messages/v1/messagesv1connect"
	"ssuspy-proto/gen/users/v1/usersv1connect"
	"time"

	"connectrpc.com/authn"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	httprateredis "github.com/go-chi/httprate-redis"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func RunWeb(repo *repository.Repository, cfg *config.StructConfig, port int64) error {
	jwtService := jwt.NewJwtService(cfg.JWTSecret)
	userService := user.NewUserService(repo)

	r := chi.NewRouter()
	r.Use(middlewares.NewCors([]string{cfg.JWTSecret}))

	rateLimitConfig := httprateredis.Config{
		Host:     cfg.Redis.Host,
		Port:     uint16(cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DBIndex:  cfg.Redis.Database,
	}

	// public
	r.Group(func(r chi.Router) {
		publicRateLimitConfig := rateLimitConfig
		publicRateLimitConfig.PrefixKey = "httprate:public"

		r.Use(httprate.Limit(
			5,
			time.Minute,
			httprate.WithKeyByIP(),
			httprateredis.WithRedisLimitCounter(&publicRateLimitConfig),
		))

		path, handler := authv1connect.NewAuthServiceHandler(
			handlers.NewAuthServer(repo, jwtService, userService),
		)
		r.Mount(path, h2c.NewHandler(handler, &http2.Server{}))
	})

	// protected
	r.Group(func(r chi.Router) {
		protectedRateLimitConfig := rateLimitConfig
		protectedRateLimitConfig.PrefixKey = "httprate:protected"

		r.Use(httprate.Limit(
			120,
			time.Minute,
			httprate.WithKeyByIP(),
			httprateredis.WithRedisLimitCounter(&protectedRateLimitConfig),
		))

		authMiddleware := authn.NewMiddleware(middlewares.NewAuthenticate(jwtService))
		r.Use(func(next http.Handler) http.Handler {
			return authMiddleware.Wrap(next)
		})

		path, handler := usersv1connect.NewUsersServiceHandler(
			handlers.NewUsersServer(repo),
		)
		r.Mount(path, h2c.NewHandler(handler, &http2.Server{}))

		path, handler = messagesv1connect.NewMessagesServiceHandler(
			handlers.NewMessagesServer(repo),
		)
		r.Mount(path, h2c.NewHandler(handler, &http2.Server{}))
	})

	return http.ListenAndServe(fmt.Sprintf(":%d", port), r)
}
