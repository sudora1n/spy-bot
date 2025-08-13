package middlewares

import (
	"net/http"

	connectcors "connectrpc.com/cors"
	"github.com/go-chi/cors"
)

func NewCors(allowedOrigins []string) func(next http.Handler) http.Handler {
	return cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   connectcors.AllowedMethods(),
		AllowedHeaders:   connectcors.AllowedHeaders(),
		ExposedHeaders:   connectcors.ExposedHeaders(),
		AllowCredentials: true,
		MaxAge:           300,
	})
}
