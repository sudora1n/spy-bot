package handlers

import (
	"ssuspy-bot/callbacks"
	"ssuspy-bot/consts"
	"ssuspy-bot/types"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func (h *Handler) HandleSettings(c *th.Context, update telego.Update) error {
	query := update.CallbackQuery
	loc := c.Value("loc").(*i18n.Localizer)
	internalUser := c.Value("internalUser").(*types.InternalUser)

	needUpdate := true
	data, err := callbacks.NewHandleSettingsDataFromString(query.Data)
	if err != nil {
		if err == callbacks.NoSettingsPartsError {
			needUpdate = false
		}
		return err
	}

	if needUpdate {
		err = h.service.UpdateUserSettings(c, internalUser.ID, data)
		if err != nil {
			return err
		}
	}

	status := map[bool]string{
		true: loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "settings.on",
		}),
		false: loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "settings.off",
		}),
	}

	messageText := loc.MustLocalize(&i18n.LocalizeConfig{
		MessageID: "settings.message",
		TemplateData: map[string]string{
			"MyDel":       status[data.ShowMyDeleted],
			"PartnerDel":  status[data.ShowPartnerDeleted],
			"MyEdit":      status[data.ShowMyEdits],
			"PartnerEdit": status[data.ShowPartnerEdits],
		},
	})

	var rows [][]telego.InlineKeyboardButton

	myDelData := *data
	myDelData.ShowMyDeleted = !myDelData.ShowMyDeleted
	rows = append(rows, tu.InlineKeyboardRow(tu.InlineKeyboardButton(
		loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "settings.myDel",
			TemplateData: map[string]bool{
				"Status": data.ShowMyDeleted,
			},
		}),
	).WithCallbackData(myDelData.ToString())))

	partnerDelData := *data
	partnerDelData.ShowMyDeleted = !partnerDelData.ShowMyDeleted
	rows = append(rows, tu.InlineKeyboardRow(tu.InlineKeyboardButton(
		loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "settings.partnerDel",
			TemplateData: map[string]bool{
				"Status": data.ShowMyDeleted,
			},
		}),
	).WithCallbackData(partnerDelData.ToString())))

	myEditData := *data
	myEditData.ShowMyDeleted = !myEditData.ShowMyDeleted
	rows = append(rows, tu.InlineKeyboardRow(tu.InlineKeyboardButton(
		loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "settings.myEdit",
			TemplateData: map[string]bool{
				"Status": data.ShowMyDeleted,
			},
		}),
	).WithCallbackData(myEditData.ToString())))

	partnerEditData := *data
	partnerEditData.ShowMyDeleted = !partnerEditData.ShowMyDeleted
	rows = append(rows, tu.InlineKeyboardRow(tu.InlineKeyboardButton(
		loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "settings.partnerEdit",
			TemplateData: map[string]bool{
				"Status": data.ShowMyDeleted,
			},
		}),
	).WithCallbackData(partnerEditData.ToString())))

	rows = append(rows, tu.InlineKeyboardRow(tu.InlineKeyboardButton(
		loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "back",
		}),
	).WithCallbackData(consts.CALLBACK_PREFIX_BACK_TO_START)))

	_, err = c.Bot().EditMessageText(c, tu.EditMessageText(
		tu.ID(internalUser.ID),
		query.Message.GetMessageID(),
		messageText,
	).WithParseMode(telego.ModeHTML).WithReplyMarkup(tu.InlineKeyboard(rows...)))
	return err
}
