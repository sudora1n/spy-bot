package handlers

import (
	"fmt"
	"ssuspy-bot/consts"
	"ssuspy-bot/telegram/callbacks"
	"ssuspy-bot/telegram/keyboard"
	"ssuspy-bot/telegram/utils"
	"ssuspy-common/repository/mongoRepository"
	"ssuspy-common/types"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func makeSettingsRows(loc *i18n.Localizer, handler string, settings []types.SettingMeta) (rows [][]telego.InlineKeyboardButton) {
	for _, s := range settings {
		label := loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: s.MessageID,
			TemplateData: map[string]bool{
				"Status": s.Status,
			},
		})

		rows = append(rows, tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(label).
				WithCallbackData(
					fmt.Sprintf("%s|%d", handler, s.Data),
				),
		))
	}

	rows = append(rows, tu.InlineKeyboardRow(
		keyboard.BuildBackButton(loc, consts.CALLBACK_PREFIX_SETTINGS),
	))

	return rows
}

func (h *Handler) HandleSettings(c *th.Context, update telego.Update) error {
	query := update.CallbackQuery
	loc := c.Value("loc").(*i18n.Localizer)
	user := c.Value("user").(*mongoRepository.User)

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
			"MyDel":       status[user.Settings.ShowMyDeleted],
			"PartnerDel":  status[user.Settings.ShowPartnerDeleted],
			"MyEdit":      status[user.Settings.ShowMyEdits],
			"PartnerEdit": status[user.Settings.ShowPartnerEdits],
		},
	})

	_, err := c.Bot().EditMessageText(c, tu.EditMessageText(
		tu.ID(user.Id),
		query.Message.GetMessageID(),
		messageText,
	).WithParseMode(telego.ModeHTML).WithReplyMarkup(
		tu.InlineKeyboard(
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton(
					loc.MustLocalize(&i18n.LocalizeConfig{
						MessageID: "settings.buttons.deleted",
					}),
				).WithCallbackData(consts.CALLBACK_PREFIX_SETTINGS_DELETED),
			),
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton(
					loc.MustLocalize(&i18n.LocalizeConfig{
						MessageID: "settings.buttons.edited",
					}),
				).WithCallbackData(consts.CALLBACK_PREFIX_SETTINGS_EDITED),
			),
			tu.InlineKeyboardRow(
				keyboard.BuildBackButton(loc, consts.CALLBACK_PREFIX_BACK_TO_START),
			),
		),
	))
	return err
}

func (h *Handler) HandleSettingsDeleted(c *th.Context, update telego.Update) error {
	query := update.CallbackQuery
	loc := c.Value("loc").(*i18n.Localizer)
	user := c.Value("user").(*mongoRepository.User)

	needUpdate := true
	data, err := callbacks.NewHandleSettingsDataFromString(query.Data)
	if err != nil {
		if err == callbacks.NoSettingsPartsError {
			needUpdate = false
		} else {
			return err
		}
	}

	if needUpdate {
		switch data {
		case consts.SETTINGS_SHOW_MY_DELETED:
			user.Settings.ShowMyDeleted = !user.Settings.ShowMyDeleted
		case consts.SETTINGS_SHOW_PARTNER_DELETED:
			user.Settings.ShowPartnerDeleted = !user.Settings.ShowPartnerDeleted
		default:
			utils.OnDataError(c, query.ID, loc)
			return fmt.Errorf("no seting found")
		}

		err = h.repo.Mongo.UpdateUserSettings(
			c,
			user.Id,
			user.Settings,
		)
		if err != nil {
			return err
		}
	}

	settings := []types.SettingMeta{
		{
			MessageID: "settings.deleted.my",
			Status:    user.Settings.ShowMyDeleted,
			Data:      consts.SETTINGS_SHOW_MY_DELETED,
		},
		{
			MessageID: "settings.deleted.partner",
			Status:    user.Settings.ShowPartnerDeleted,
			Data:      consts.SETTINGS_SHOW_PARTNER_DELETED,
		},
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
		MessageID: "settings.deleted.message",
		TemplateData: map[string]string{
			"My":      status[user.Settings.ShowMyDeleted],
			"Partner": status[user.Settings.ShowPartnerDeleted],
		},
	})

	_, err = c.Bot().EditMessageText(c, tu.EditMessageText(
		tu.ID(user.Id),
		query.Message.GetMessageID(),
		messageText,
	).WithParseMode(telego.ModeHTML).WithReplyMarkup(tu.InlineKeyboard(makeSettingsRows(loc, consts.CALLBACK_PREFIX_SETTINGS_DELETED, settings)...)))
	return err
}

func (h *Handler) HandleSettingsEdited(c *th.Context, update telego.Update) error {
	query := update.CallbackQuery
	loc := c.Value("loc").(*i18n.Localizer)
	user := c.Value("user").(*mongoRepository.User)

	needUpdate := true
	data, err := callbacks.NewHandleSettingsDataFromString(query.Data)
	if err != nil {
		if err == callbacks.NoSettingsPartsError {
			needUpdate = false
		} else {
			return err
		}
	}

	if needUpdate {
		switch data {
		case consts.SETTINGS_SHOW_MY_EDITS:
			user.Settings.ShowMyEdits = !user.Settings.ShowMyEdits
		case consts.SETTINGS_SHOW_PARTNER_EDITS:
			user.Settings.ShowPartnerEdits = !user.Settings.ShowPartnerEdits
		default:
			utils.OnDataError(c, query.ID, loc)
			return fmt.Errorf("no seting found")
		}

		err = h.repo.Mongo.UpdateUserSettings(
			c,
			user.Id,
			user.Settings,
		)
		if err != nil {
			return err
		}
	}

	settings := []types.SettingMeta{
		{
			MessageID: "settings.edited.my",
			Status:    user.Settings.ShowMyEdits,
			Data:      consts.SETTINGS_SHOW_MY_EDITS,
		},
		{
			MessageID: "settings.edited.partner",
			Status:    user.Settings.ShowPartnerEdits,
			Data:      consts.SETTINGS_SHOW_PARTNER_EDITS,
		},
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
		MessageID: "settings.edited.message",
		TemplateData: map[string]string{
			"My":      status[user.Settings.ShowMyEdits],
			"Partner": status[user.Settings.ShowPartnerEdits],
		},
	})

	_, err = c.Bot().EditMessageText(c, tu.EditMessageText(
		tu.ID(user.Id),
		query.Message.GetMessageID(),
		messageText,
	).WithParseMode(telego.ModeHTML).WithReplyMarkup(tu.InlineKeyboard(makeSettingsRows(loc, consts.CALLBACK_PREFIX_SETTINGS_EDITED, settings)...)))
	return err
}
