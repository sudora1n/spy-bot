package utils

import (
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func GetBackButton(loc *i18n.Localizer, data string) telego.InlineKeyboardButton {
	return tu.InlineKeyboardButton(
		loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "back",
		}),
	).WithCallbackData(data)
}
