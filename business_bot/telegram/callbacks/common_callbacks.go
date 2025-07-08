package callbacks

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var NoSettingsPartsError = errors.New("wrong number of parameters of settings data")

func NewHandleSettingsDataFromString(s string) (setting int, err error) {
	parts := strings.Split(s, "|")
	if len(parts) != 2 {
		return -1, NoSettingsPartsError
	}

	setting, err = strconv.Atoi(parts[1])
	if err != nil {
		return -1, fmt.Errorf("failed to convert mask from string: %v", err)
	}

	return setting, nil
}
