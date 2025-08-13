package callbacks

import (
	"fmt"
	"ssuspy-bot/consts"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type HandleEditedData struct {
	DataID primitive.ObjectID
	ChatID int64
}

type HandleEditedDataType int

const (
	HandleEditedDataTypeLog HandleEditedDataType = iota
	HandleEditedDataTypeFiles
)

func (h HandleEditedData) ToString(dataType HandleEditedDataType) string {
	prefix := ""
	switch dataType {
	case HandleEditedDataTypeFiles:
		prefix = consts.CALLBACK_PREFIX_EDITED_FILES
	case HandleEditedDataTypeLog:
		prefix = consts.CALLBACK_PREFIX_EDITED_LOG
	default:
		return ""
	}
	return fmt.Sprintf("%s|%d|%s", prefix, h.ChatID, h.DataID.Hex())
}

func NewHandleEditedLogDataFromString(s string) (data *HandleEditedData, err error) {
	expectedLen := 3

	parts := strings.Split(s, "|")
	if len(parts) != expectedLen {
		return nil, fmt.Errorf("wrong number of parameters: expected %d, received %d", expectedLen, len(parts))
	}

	data = &HandleEditedData{}
	data.ChatID, err = strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to convert ChatID: %v", err)
	}

	data.DataID, err = primitive.ObjectIDFromHex(parts[2])
	if err != nil {
		return nil, fmt.Errorf("failed to convert DataID: %v", err)
	}

	return data, nil
}
