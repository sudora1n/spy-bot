package handlers

import (
	"ssuspy-bot/callbacks"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
)

func HandleSettings(c *th.Context, update telego.Update) error {
	query := update.CallbackQuery
	// loc := c.Value("loc").(*i18n.Localizer)

	_, err := callbacks.NewHandleSettingsDataFromString(query.Data)
	if err != nil {
		if err == callbacks.NoSettingsPartsError {
			//
		}
		return err
	}

	return nil
}
