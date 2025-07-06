package handlers

import (
	"html"
	"ssuspy-bot/consts"
	"ssuspy-bot/repository"
	"ssuspy-bot/telegram/utils"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/rs/zerolog"
)

const (
	LOVE_COMMAND   = ".love"
	LOVEUA_COMMAND = ".loveua"
	LOVERU_COMMAND = ".loveru"
)

func (h *Handler) HandleUserHelp(c *th.Context, update telego.Update) error {
	message := update.BusinessMessage
	loc := c.Value("loc").(*i18n.Localizer)
	iUser := c.Value("iUser").(*repository.IUser)
	rights := c.Value("rights").(*telego.BusinessBotRights)
	connection := c.Value("userConnection").(*repository.BotUserBusinessConnection)

	if !rights.CanReply {
		return utils.OnCantReply(c, loc, iUser.User.ID, ".help")
	}

	_, err := c.Bot().EditMessageText(
		c,
		tu.EditMessageText(tu.ID(message.Chat.ID), message.MessageID, loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "userHandlers.help",
		})).WithBusinessConnectionID(connection.ID).WithParseMode(telego.ModeHTML),
	)
	return err
}

func (h *Handler) HandleUserAnimation(c *th.Context, update telego.Update) error { // .a // .anim
	message := update.BusinessMessage
	loc := c.Value("loc").(*i18n.Localizer)
	log := c.Value("log").(*zerolog.Logger)
	iUser := c.Value("iUser").(*repository.IUser)
	rights := c.Value("rights").(*telego.BusinessBotRights)
	connection := c.Value("userConnection").(*repository.BotUserBusinessConnection)

	var text string
	parts := strings.SplitN(message.Text, " ", 2)
	if len(parts) > 1 {
		text = parts[1]
	}

	if !rights.CanReply {
		return utils.OnCantReply(c, loc, iUser.User.ID, ".(a|anim)")
	}

	frames := utils.GenerateBatchAnimationFrames(text, 10)
	if len(frames) == 0 {
		log.Warn().Msg("zero frames")
		return nil
	}

	var prev string
	for _, frame := range frames {
		frame = strings.TrimSpace(frame)
		if text == frame || frame == prev {
			continue
		}

		_, err := c.Bot().EditMessageText(
			c,
			tu.EditMessageText(
				tu.ID(message.Chat.ID),
				message.MessageID,
				html.EscapeString(frame),
			).WithBusinessConnectionID(connection.ID).WithParseMode(telego.ModeHTML),
		)
		if err != nil {
			return err
		}

		prev = frame

		time.Sleep(time.Millisecond * 400)
	}

	log.Debug().Str("userHandler", "anim").Msg(text)
	if text != prev {
		_, err := c.Bot().EditMessageText(
			c,
			tu.EditMessageText(
				tu.ID(message.Chat.ID),
				message.MessageID,
				html.EscapeString(text),
			).WithBusinessConnectionID(connection.ID).WithParseMode(telego.ModeHTML),
		)
		return err
	}

	return nil
}

func (h *Handler) HandleUserLove(c *th.Context, update telego.Update) error {
	message := update.BusinessMessage
	loc := c.Value("loc").(*i18n.Localizer)
	iUser := c.Value("iUser").(*repository.IUser)
	rights := c.Value("rights").(*telego.BusinessBotRights)
	connection := c.Value("userConnection").(*repository.BotUserBusinessConnection)

	var text string
	parts := strings.SplitN(message.Text, " ", 2)
	if len(parts) > 0 {
		text = parts[0]
	}

	var (
		repeat  = 5
		command = LOVE_COMMAND
		frames  = consts.JustLove
	)
	switch {
	case strings.HasSuffix(text, "ua"):
		command = LOVEUA_COMMAND
	case strings.HasSuffix(text, "ru"):
		repeat = 3
		command = LOVERU_COMMAND
	}

	if !rights.CanReply {
		return utils.OnCantReply(c, loc, iUser.User.ID, command)
	}

	for range repeat {
		for _, frame := range frames {
			_, err := c.Bot().EditMessageText(
				c,
				tu.EditMessageText(
					tu.ID(message.Chat.ID),
					message.MessageID,
					frame,
				).WithBusinessConnectionID(connection.ID),
			)
			if err != nil {
				return err
			}

			time.Sleep(400 * time.Millisecond)
		}
	}

	return nil
}
