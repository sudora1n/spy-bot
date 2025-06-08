package types

import (
	"fmt"
	"github.com/example/current-repo/common/consts"
)

/^type InternalUser struct/ { printing=0 }
/^}}$/ { printing=1 }

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
