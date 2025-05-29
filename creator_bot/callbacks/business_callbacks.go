package callbacks

import (
	"fmt"
	"ssuspy-creator-bot/types"
	"strconv"
	"strings"
)

func NewHandleBotItemFromString(s string) (data *types.HandleBotItem, err error) {
	expectedLen := 2

	parts := strings.Split(s, "|")
	if len(parts) != expectedLen {
		return nil, fmt.Errorf("wrong number of parameters: expected %d, received %d", expectedLen, len(parts))
	}

	data = &types.HandleBotItem{}

	data.BotID, err = strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to convert BotID: %v", err)
	}

	return data, nil
}
