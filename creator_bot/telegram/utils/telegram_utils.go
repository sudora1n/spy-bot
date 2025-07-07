package utils

import (
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func OnDataError(c *th.Context, queryID string, loc *i18n.Localizer) {
	c.Bot().AnswerCallbackQuery(c, tu.CallbackQuery(queryID).WithText(
		loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "errors.couldNotRetrieveData",
		}),
	))
}
