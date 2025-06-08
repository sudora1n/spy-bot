package utils

import (
	"time"

	commonProm "github.com/example/current-repo/common/prom"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/rs/zerolog/log"
)

// WithProm is a middleware utility that wraps a handler with Prometheus metrics.
// It assumes that commonProm metrics (RequestsTotal, ProcessingTime, ErrorsTotal) are initialized.
func WithProm(name string, handler th.Handler) th.Handler {
	return func(c *th.Context, update telego.Update) error {
		start := time.Now()
		if commonProm.RequestsTotal != nil {
			commonProm.RequestsTotal.WithLabelValues(name).Inc()
		} else {
			log.Warn().Str("metric_handler", name).Msg("commonProm.RequestsTotal is nil, skipping metric increment")
		}

		defer func() {
			duration := time.Since(start).Seconds()
			if commonProm.ProcessingTime != nil {
				commonProm.ProcessingTime.WithLabelValues(name).Observe(duration)
			} else {
				log.Warn().Str("metric_handler", name).Msg("commonProm.ProcessingTime is nil, skipping metric observation")
			}
		}()

		err := handler(c, update)
		if err != nil {
			if commonProm.ErrorsTotal != nil {
				commonProm.ErrorsTotal.WithLabelValues(name).Inc()
			} else {
				log.Warn().Str("metric_handler", name).Err(err).Msg("commonProm.ErrorsTotal is nil, skipping error metric increment")
			}
		}
		return err
	}
}

// OnDataError sends a standardized callback query answer when data retrieval fails.
// It attempts to use a localizer from context or falls back to a default message.
func OnDataError(c *th.Context, queryID string, loc *i18n.Localizer) {
	if c == nil || c.Bot() == nil {
		log.Error().Str("queryID", queryID).Msg("Context or Bot is nil in OnDataError")
		return
	}

	var messageText string
	if loc != nil {
		messageText = loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "errors.couldNotRetrieveData", // Assuming this message ID exists
		})
	} else {
		// Attempt to get localizer from context if not directly provided
		locVal := c.Value("loc")
		if l, ok := locVal.(*i18n.Localizer); ok && l != nil {
			loc = l
			messageText = loc.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "errors.couldNotRetrieveData",
			})
		} else {
			log.Warn().Str("queryID", queryID).Msg("Localizer is nil and not found in context for OnDataError. Using default error message.")
			messageText = "Error: Could not retrieve data." // Default non-localized message
		}
	}

	errAns := c.Bot().AnswerCallbackQuery(c, tu.CallbackQuery(queryID).WithText(messageText))
	if errAns != nil {
		log.Error().Err(errAns).Str("queryID", queryID).Msg("Failed to answer callback query in OnDataError")
	}
}
