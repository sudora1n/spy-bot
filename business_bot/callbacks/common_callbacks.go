package callbacks

import (
	"errors"
	"fmt"
	"ssuspy-bot/types"
	"strconv"
	"strings"
)

var NoSettingsPartsError = errors.New("wrong number of parameters of settings data")

func NewHandleSettingsDataFromString(s string) (data *types.HandleSettingsData, err error) {
	parts := strings.Split(s, "|")
	if len(parts) != 2 {
		return nil, NoSettingsPartsError
	}

	maskInt, err := strconv.ParseUint(parts[1], 2, 8)
	if err != nil {
		return nil, fmt.Errorf("failed to convert mask from string: %v", err)
	}

	mask := uint8(maskInt)

	data = &types.HandleSettingsData{
		ShowMyDeleted:      mask&types.BitShowMyEdits != 0,
		ShowPartnerDeleted: mask&types.BitShowPartnerEdits != 0,
		ShowMyEdits:        mask&types.BitShowMyDeleted != 0,
		ShowPartnerEdits:   mask&types.BitShowPartnerDel != 0,
	}

	return data, nil
}
