package callbacks

import (
	"fmt"
	"ssuspy-bot/consts"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type HandleBusinessData struct {
	DataID primitive.ObjectID
	ChatID int64
}

type HandleBusinessDataType int

const (
	HandleBusinessDataTypeDeleted HandleBusinessDataType = iota
)

func (h HandleBusinessData) ToString(dataType HandleBusinessDataType) string {
	prefix := ""

	switch dataType {
	case HandleBusinessDataTypeDeleted:
		prefix = consts.CALLBACK_PREFIX_DELETED
	default:
		return ""
	}

	return fmt.Sprintf("%s|%s|%d", prefix, h.DataID.Hex(), h.ChatID)
}

func NewHandleBusinessDataFromString(s string) (data *HandleBusinessData, err error) {
	expectedLen := 3

	parts := strings.Split(s, "|")
	if len(parts) != expectedLen {
		return nil, fmt.Errorf("wrong number of parameters: expected %d, received %d", expectedLen, len(parts))
	}

	data = &HandleBusinessData{}

	data.DataID, err = primitive.ObjectIDFromHex(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to convert DataID: %v", err)
	}

	data.ChatID, err = strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to convert ChatID: %v", err)
	}

	return data, nil
}
