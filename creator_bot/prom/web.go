package prom

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func RunMetrics() error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	return http.ListenAndServe(":8081", mux)
}
