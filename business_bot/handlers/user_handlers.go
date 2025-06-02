package handlers

import (
	"ssuspy-bot/repository"
	"ssuspy-bot/utils"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/rs/zerolog"
)

func (h *Handler) HandleUserHelp(c *th.Context, update telego.Update) error {
	// go h.HandleMessage(c, update)

	message := update.BusinessMessage
	log := c.Value("log").(*zerolog.Logger)
	loc := c.Value("loc").(*i18n.Localizer)
	iUser := c.Value("iUser").(*repository.IUser)

	connection := iUser.BotUser.GetUserCurrentConnection()
	rights, err := utils.GetBusinessRights(c, connection)
	if err != nil {
		log.Warn().Err(err).Msg("failed get business connection")
		return err
	}
	if !rights.CanReply {
		_, err = c.Bot().SendMessage(
			c,
			tu.Message(tu.ID(iUser.User.ID), loc.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "errors.userHandlers.noCanReply",
				TemplateData: map[string]string{
					"Command": "help",
				},
			})),
		)
		return err
	}

	_, err = c.Bot().EditMessageText(
		c,
		tu.EditMessageText(tu.ID(message.Chat.ID), message.MessageID, loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "userHandlers.help",
		})).WithBusinessConnectionID(connection.ID).WithParseMode(telego.ModeHTML),
	)
	return err
}
