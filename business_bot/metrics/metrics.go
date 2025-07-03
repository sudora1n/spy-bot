package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	RequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bot_requests_total",
			Help: "Total number of requests per handler",
		},
		[]string{"handler"},
	)

	ErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bot_errors_total",
			Help: "Total number of errors per handler",
		},
		[]string{"handler"},
	)

	PanicsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "bot_panics_total",
			Help: "Total number of panics",
		},
	)

	ProcessingTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "bot_handler_duration_seconds",
			Help:    "Duration of handler execution",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"handler"},
	)
)

func init() {
	prometheus.MustRegister(RequestsTotal)
	prometheus.MustRegister(ErrorsTotal)
	prometheus.MustRegister(PanicsTotal)
	prometheus.MustRegister(ProcessingTime)
}
