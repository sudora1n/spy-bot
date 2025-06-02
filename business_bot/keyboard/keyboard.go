package keyboard

import (
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func BuildOnNewReplyMarkup(loc *i18n.Localizer) [][]telego.InlineKeyboardButton {
	return [][]telego.InlineKeyboardButton{
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(
				loc.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "start.onNew.buttons.open",
				}),
			).WithURL(
				loc.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "start.onNew.link",
				}),
			),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(
				loc.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "start.onNew.buttons.settings",
				}),
			).WithURL("tg://settings"),
		),
	}
}
