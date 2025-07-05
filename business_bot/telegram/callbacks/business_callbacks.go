package callbacks

import (
	"fmt"
	"ssuspy-bot/types"
	"strconv"
	"strings"
)

func NewHandleBusinessDataFromString(s string) (data *types.HandleBusinessData, err error) {
	expectedLen := 3

	parts := strings.Split(s, "|")
	if len(parts) != expectedLen {
		return nil, fmt.Errorf("wrong number of parameters: expected %d, received %d", expectedLen, len(parts))
	}

	data = &types.HandleBusinessData{}

	data.DataID, err = strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to convert DataID: %v", err)
	}

	data.ChatID, err = strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to convert ChatID: %v", err)
	}

	return data, nil
}
