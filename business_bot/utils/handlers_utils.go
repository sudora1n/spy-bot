package utils

import (
	"ssuspy-bot/prom"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
)

func WithProm(name string, handler th.Handler) th.Handler {
	return func(c *th.Context, update telego.Update) error {
		start := time.Now()
		prom.RequestsTotal.WithLabelValues(name).Inc()

		defer func() {
			duration := time.Since(start).Seconds()
			prom.ProcessingTime.WithLabelValues(name).Observe(duration)
		}()

		err := handler(c, update)
		if err != nil {
			prom.ErrorsTotal.WithLabelValues(name).Inc()
		}

		return err
	}
}
