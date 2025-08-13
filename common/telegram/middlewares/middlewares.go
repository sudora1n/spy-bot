package middleware

import (
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
)

func AutoRespond(c *th.Context, update telego.Update) error {
	if update.CallbackQuery != nil {
		defer func() {
			c.Bot().AnswerCallbackQuery(c, tu.CallbackQuery(update.CallbackQuery.ID))
		}()
	}

	return c.Next(update)
}
