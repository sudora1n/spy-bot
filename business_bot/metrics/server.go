package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

func RunMetrics(mux *http.ServeMux) {
	mux.Handle("/metrics", promhttp.Handler())

	if err := http.ListenAndServe(":8080", mux); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("web server server down")
	}
}
