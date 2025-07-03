package handlers

import (
	"fmt"
	"ssuspy-bot/callbacks"
	"ssuspy-bot/consts"
	"ssuspy-bot/repository"
	"ssuspy-bot/telegram/keyboard"
	"ssuspy-bot/telegram/utils"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type settingMeta struct {
	messageID string
	status    bool
	data      int
}

func makeSettingsRows(loc *i18n.Localizer, handler string, settings []settingMeta) (rows [][]telego.InlineKeyboardButton) {
	for _, s := range settings {
		label := loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: s.messageID,
			TemplateData: map[string]bool{
				"Status": s.status,
			},
		})

		rows = append(rows, tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(label).
				WithCallbackData(
					fmt.Sprintf("%s|%d", handler, s.data),
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
	iUser := c.Value("iUser").(*repository.IUser)

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
			"MyDel":       status[iUser.User.Settings.ShowMyDeleted],
			"PartnerDel":  status[iUser.User.Settings.ShowPartnerDeleted],
			"MyEdit":      status[iUser.User.Settings.ShowMyEdits],
			"PartnerEdit": status[iUser.User.Settings.ShowPartnerEdits],
		},
	})

	_, err := c.Bot().EditMessageText(c, tu.EditMessageText(
		tu.ID(iUser.User.ID),
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
	iUser := c.Value("iUser").(*repository.IUser)

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
			iUser.User.Settings.ShowMyDeleted = !iUser.User.Settings.ShowMyDeleted
		case consts.SETTINGS_SHOW_PARTNER_DELETED:
			iUser.User.Settings.ShowPartnerDeleted = !iUser.User.Settings.ShowPartnerDeleted
		default:
			utils.OnDataError(c, query.ID, loc)
			return fmt.Errorf("no seting found")
		}

		err = h.service.UpdateUserSettings(
			c,
			iUser.User.ID,
			iUser.User.Settings,
		)
		if err != nil {
			return err
		}
	}

	settings := []settingMeta{
		{
			messageID: "settings.deleted.my",
			status:    iUser.User.Settings.ShowMyDeleted,
			data:      consts.SETTINGS_SHOW_MY_DELETED,
		},
		{
			messageID: "settings.deleted.partner",
			status:    iUser.User.Settings.ShowPartnerDeleted,
			data:      consts.SETTINGS_SHOW_PARTNER_DELETED,
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
			"My":      status[iUser.User.Settings.ShowMyDeleted],
			"Partner": status[iUser.User.Settings.ShowPartnerDeleted],
		},
	})

	_, err = c.Bot().EditMessageText(c, tu.EditMessageText(
		tu.ID(iUser.User.ID),
		query.Message.GetMessageID(),
		messageText,
	).WithParseMode(telego.ModeHTML).WithReplyMarkup(tu.InlineKeyboard(makeSettingsRows(loc, consts.CALLBACK_PREFIX_SETTINGS_DELETED, settings)...)))
	return err
}

func (h *Handler) HandleSettingsEdited(c *th.Context, update telego.Update) error {
	query := update.CallbackQuery
	loc := c.Value("loc").(*i18n.Localizer)
	iUser := c.Value("iUser").(*repository.IUser)

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
			iUser.User.Settings.ShowMyEdits = !iUser.User.Settings.ShowMyEdits
		case consts.SETTINGS_SHOW_PARTNER_EDITS:
			iUser.User.Settings.ShowPartnerEdits = !iUser.User.Settings.ShowPartnerEdits
		default:
			utils.OnDataError(c, query.ID, loc)
			return fmt.Errorf("no seting found")
		}

		err = h.service.UpdateUserSettings(
			c,
			iUser.User.ID,
			iUser.User.Settings,
		)
		if err != nil {
			return err
		}
	}

	settings := []settingMeta{
		{
			messageID: "settings.edited.my",
			status:    iUser.User.Settings.ShowMyEdits,
			data:      consts.SETTINGS_SHOW_MY_EDITS,
		},
		{
			messageID: "settings.edited.partner",
			status:    iUser.User.Settings.ShowPartnerEdits,
			data:      consts.SETTINGS_SHOW_PARTNER_EDITS,
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
			"My":      status[iUser.User.Settings.ShowMyEdits],
			"Partner": status[iUser.User.Settings.ShowPartnerEdits],
		},
	})

	_, err = c.Bot().EditMessageText(c, tu.EditMessageText(
		tu.ID(iUser.User.ID),
		query.Message.GetMessageID(),
		messageText,
	).WithParseMode(telego.ModeHTML).WithReplyMarkup(tu.InlineKeyboard(makeSettingsRows(loc, consts.CALLBACK_PREFIX_SETTINGS_EDITED, settings)...)))
	return err
}
