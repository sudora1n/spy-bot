package utils

import (
	"ssuspy-bot/metrics"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
)

func WithProm(name string, handler th.Handler) th.Handler {
	return func(c *th.Context, update telego.Update) error {
		start := time.Now()
		metrics.RequestsTotal.WithLabelValues(name).Inc()

		defer func() {
			duration := time.Since(start).Seconds()
			metrics.ProcessingTime.WithLabelValues(name).Observe(duration)
		}()

		err := handler(c, update)
		if err != nil {
			metrics.ErrorsTotal.WithLabelValues(name).Inc()
		}

		return err
	}
}
