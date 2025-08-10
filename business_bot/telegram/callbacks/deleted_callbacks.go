package callbacks

import (
	"fmt"
	"ssuspy-bot/consts"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type HandleDeletedPaginationData struct {
	DataID           primitive.ObjectID
	ChatID           int64
	Offset           int
	TypeOfPagination string
}

func (h HandleDeletedPaginationData) ToString() string {
	return fmt.Sprintf("%s|%s|%d|%d|%s", consts.CALLBACK_PREFIX_DELETED, h.DataID.Hex(), h.ChatID, h.Offset, h.TypeOfPagination)
}

func NewHandleDeletedPaginationDataFromString(s string) (*HandleDeletedPaginationData, error) {
	expectedLen := 5

	parts := strings.Split(s, "|")
	if len(parts) != expectedLen {
		return nil, fmt.Errorf("wrong number of parameters: expected %d, received %d", expectedLen, len(parts))
	}

	dataID, err := primitive.ObjectIDFromHex(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to convert DataID: %v", err)
	}

	chatID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to convert ChatID: %v", err)
	}

	offset, err := strconv.Atoi(parts[3])
	if err != nil {
		return nil, fmt.Errorf("failed to convert Offset: %v", err)
	}

	return &HandleDeletedPaginationData{
		DataID:           dataID,
		ChatID:           chatID,
		Offset:           offset,
		TypeOfPagination: parts[4],
	}, nil
}

type HandleDeletedLogData struct {
	DataID primitive.ObjectID
	ChatID int64
	Offset int
}

func (h HandleDeletedLogData) ToString() string {
	return fmt.Sprintf("%s|%s|%d|%d", consts.CALLBACK_PREFIX_DELETED_LOG, h.DataID.Hex(), h.ChatID, h.Offset)
}

func NewHandleDeletedLogDataFromString(s string) (*HandleDeletedLogData, error) {
	expectedLen := 4

	parts := strings.Split(s, "|")
	if len(parts) != expectedLen {
		return nil, fmt.Errorf("wrong number of parameters: expected %d, received %d", expectedLen, len(parts))
	}

	dataID, err := primitive.ObjectIDFromHex(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to convert DataID: %v", err)
	}

	chatID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to convert ChatID: %v", err)
	}

	offset, err := strconv.Atoi(parts[3])
	if err != nil {
		return nil, fmt.Errorf("failed to convert Offset: %v", err)
	}

	return &HandleDeletedLogData{
		DataID: dataID,
		ChatID: chatID,
		Offset: offset,
	}, nil
}

type HandleDeletedMessageData struct {
	MessageID  int
	ChatID     int64
	DataID     primitive.ObjectID
	BackOffset int
}

type HandleDeletedMessageDataType int

const (
	HandleDeletedMessageDataTypeDetails HandleDeletedMessageDataType = iota
	HandleDeletedMessageDataTypeMessage
)

func (h HandleDeletedMessageData) ToString(dataType HandleDeletedMessageDataType) string {
	prefix := ""

	switch dataType {
	case HandleDeletedMessageDataTypeDetails:
		prefix = consts.CALLBACK_PREFIX_DELETED_DETAILS
	case HandleDeletedMessageDataTypeMessage:
		prefix = consts.CALLBACK_PREFIX_DELETED_MESSAGE
	default:
		return ""
	}

	return fmt.Sprintf("%s|%d|%d|%s|%d", prefix, h.MessageID, h.ChatID, h.DataID.Hex(), h.BackOffset)
}

func NewHandleDeletedMessageDataFromString(s string) (data *HandleDeletedMessageData, err error) {
	expectedLen := 5

	parts := strings.Split(s, "|")
	if len(parts) != expectedLen {
		return nil, fmt.Errorf("wrong number of parameters: expected %d, received %d", expectedLen, len(parts))
	}

	data = &HandleDeletedMessageData{}
	data.MessageID, err = strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to convert MessageID: %v", err)
	}

	data.ChatID, err = strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to convert ChatID: %v", err)
	}

	data.DataID, err = primitive.ObjectIDFromHex(parts[3])
	if err != nil {
		return nil, fmt.Errorf("failed to convert DataID: %v", err)
	}

	data.BackOffset, err = strconv.Atoi(parts[4])
	if err != nil {
		return nil, fmt.Errorf("failed to convert BackOffset: %v", err)
	}

	return data, nil
}

type HandleDeletedFilesDataType int

const (
	HandleDeletedFilesDataTypeMessage HandleDeletedFilesDataType = iota
	HandleDeletedFilesDataTypeData
)

type HandleDeletedFilesData struct {
	MessageID int
	ChatID    int64
	DataID    primitive.ObjectID
	Type      HandleDeletedFilesDataType
}

func (h HandleDeletedFilesData) ToString() string {
	switch h.Type {
	case HandleDeletedFilesDataTypeMessage:
		return fmt.Sprintf("%s|%d|%d|%d", consts.CALLBACK_PREFIX_DELETED_FILES, h.MessageID, h.ChatID, HandleDeletedFilesDataTypeMessage)
	case HandleDeletedFilesDataTypeData:
		return fmt.Sprintf("%s|%s|%d|%d", consts.CALLBACK_PREFIX_DELETED_FILES, h.DataID.Hex(), h.ChatID, HandleDeletedFilesDataTypeData)
	default:
		return ""
	}
}

func NewHandleDeletedFilesFromString(s string) (*HandleDeletedFilesData, error) {
	expectedLen := 4

	parts := strings.Split(s, "|")
	if len(parts) != expectedLen {
		return nil, fmt.Errorf("wrong number of parameters: expected %d, received %d", expectedLen, len(parts))
	}

	var data HandleDeletedFilesData
	parsedDataType, err := strconv.Atoi(parts[3])
	if err != nil {
		return nil, fmt.Errorf("failed to convert Type: %v", err)
	}
	data.Type = HandleDeletedFilesDataType(parsedDataType)

	switch data.Type {
	case HandleDeletedFilesDataTypeMessage:
		data.MessageID, err = strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("failed to convert MessageID: %v", err)
		}
	case HandleDeletedFilesDataTypeData:
		data.DataID, err = primitive.ObjectIDFromHex(parts[1])
		if err != nil {
			return nil, fmt.Errorf("failed to convert DataID: %v", err)
		}
	default:
		return nil, fmt.Errorf("unknown type:  %s", parts[3])
	}

	data.ChatID, err = strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to convert ChatID: %v", err)
	}

	return &data, nil
}
