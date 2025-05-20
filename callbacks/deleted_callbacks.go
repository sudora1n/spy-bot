package callbacks

import (
	"fmt"
	"ssuspy-bot/types"
	"strconv"
	"strings"
)

func NewHandleDeletedPaginationDataFromString(s string) (*types.HandleDeletedPaginationData, error) {
	expectedLen := 5

	parts := strings.Split(s, "|")
	if len(parts) != expectedLen {
		return nil, fmt.Errorf("wrong number of parameters: expected %d, received %d", expectedLen, len(parts))
	}

	dataID, err := strconv.ParseInt(parts[1], 10, 64)
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

	return &types.HandleDeletedPaginationData{
		DataID:           dataID,
		ChatID:           chatID,
		Offset:           offset,
		TypeOfPagination: parts[4],
	}, nil
}

func NewHandleDeletedLogDataFromString(s string) (*types.HandleDeletedLogData, error) {
	expectedLen := 3

	parts := strings.Split(s, "|")
	if len(parts) != expectedLen {
		return nil, fmt.Errorf("wrong number of parameters: expected %d, received %d", expectedLen, len(parts))
	}

	dataID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to convert DataID: %v", err)
	}

	chatID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to convert ChatID: %v", err)
	}

	return &types.HandleDeletedLogData{
		DataID: dataID,
		ChatID: chatID,
	}, nil
}

func NewHandleDeletedMessageDataFromString(s string) (data *types.HandleDeletedMessageData, err error) {
	expectedLen := 4

	parts := strings.Split(s, "|")
	if len(parts) != expectedLen {
		return nil, fmt.Errorf("wrong number of parameters: expected %d, received %d", expectedLen, len(parts))
	}

	data = &types.HandleDeletedMessageData{}
	data.MessageID, err = strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to convert MessageID: %v", err)
	}

	data.ChatID, err = strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to convert ChatID: %v", err)
	}

	data.DataID, err = strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to convert DataID: %v", err)
	}

	return data, nil
}

func NewHandleDeletedFilesFromString(s string) (*types.HandleDeletedFilesData, error) {
	expectedLen := 4

	parts := strings.Split(s, "|")
	if len(parts) != expectedLen {
		return nil, fmt.Errorf("wrong number of parameters: expected %d, received %d", expectedLen, len(parts))
	}

	var data types.HandleDeletedFilesData
	parsedDataType, err := strconv.Atoi(parts[3])
	if err != nil {
		return nil, fmt.Errorf("failed to convert Type: %v", err)
	}
	data.Type = types.HandleDeletedFilesDataType(parsedDataType)

	switch data.Type {
	case types.HandleDeletedFilesDataTypeMessage:
		data.MessageID, err = strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("failed to convert MessageID: %v", err)
		}
	case types.HandleDeletedFilesDataTypeData:
		data.DataID, err = strconv.ParseInt(parts[1], 10, 64)
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
