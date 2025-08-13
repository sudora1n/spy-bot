package middlewares

import (
	"context"
	"net/http"
	"ssuspy-api/service/jwt"

	"connectrpc.com/authn"
)

func NewAuthenticate(jwtService *jwt.JwtService) func(context.Context, *http.Request) (any, error) {
	return func(_ context.Context, req *http.Request) (any, error) {
		cookie, err := req.Cookie("access_token")
		if err != nil {
			return nil, authn.Errorf("missing token")
		}

		return jwtService.ValidateJwt(cookie.Value)
	}
}
