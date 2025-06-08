package prom

import (
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	RequestsTotal  *prometheus.CounterVec
	ErrorsTotal    *prometheus.CounterVec
	PanicsTotal    prometheus.Counter
	ProcessingTime *prometheus.HistogramVec
)

var registerMetricsOnce sync.Once

// InitProm initializes and registers Prometheus metrics with a given prefix.
// This function should be called once from each bot's main.go or equivalent setup.
func InitProm(metricPrefix string) {
	registerMetricsOnce.Do(func() {
		RequestsTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: fmt.Sprintf("%s_requests_total", metricPrefix),
				Help: "Total number of requests per handler",
			},
			[]string{"handler"},
		)

		ErrorsTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: fmt.Sprintf("%s_errors_total", metricPrefix),
				Help: "Total number of errors per handler",
			},
			[]string{"handler"},
		)

		PanicsTotal = prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: fmt.Sprintf("%s_panics_total", metricPrefix),
				Help: "Total number of panics",
			},
		)

		ProcessingTime = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    fmt.Sprintf("%s_handler_duration_seconds", metricPrefix),
				Help:    "Duration of handler execution",
				Buckets: prometheus.DefBuckets, // or make buckets configurable if needed
			},
			[]string{"handler"},
		)

		prometheus.MustRegister(RequestsTotal)
		prometheus.MustRegister(ErrorsTotal)
		prometheus.MustRegister(PanicsTotal)
		prometheus.MustRegister(ProcessingTime)
	})
}
