package keyboard

import (
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func BuildInstructionsKeyboardRows(loc *i18n.Localizer) []telego.InlineKeyboardButton {
	return []telego.InlineKeyboardButton{
		tu.InlineKeyboardButton(
			loc.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "instruction.open",
			}),
		).WithURL(
			loc.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "instruction.link",
			}),
		),
		tu.InlineKeyboardButton(
			loc.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "instruction.botfather",
			}),
		).WithURL("https://t.me/botfather"),
	}
}

func ButtonsToRows(btns []telego.InlineKeyboardButton) (rows [][]telego.InlineKeyboardButton) {
	for _, btn := range btns {
		rows = append(rows, tu.InlineKeyboardRow(btn))
	}
	return rows
}
