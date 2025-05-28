package types

import (
	"fmt"
	"ssuspy-creator-bot/consts"
)

type InternalUser struct {
	ID           int64
	FirstName    string // optional
	LastName     string // optional
	LanguageCode string
	SendMessages bool
}

type HandleBotItem struct {
	BotID int64
}

func (h *HandleBotItem) String() string {
	return fmt.Sprintf("%s|%d", consts.CALLBACK_PREFIX_BOT_ITEM, h.BotID)
}

type HandleBotRemove struct {
	BotID int64
}

func (h *HandleBotRemove) String() string {
	return fmt.Sprintf("%s|%d", consts.CALLBACK_PREFIX_BOT_REMOVE, h.BotID)
}
