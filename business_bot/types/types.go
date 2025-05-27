package types

import (
	"fmt"
	"ssuspy-bot/consts"
)

type InternalUser struct {
	ID                   int64
	FirstName            string // optional
	LastName             string // optional
	LanguageCode         string
	SendMessages         bool
	BusinessConnectionID string // optional
}

type MediaItemProcess struct {
	Type     string
	FileID   string
	FileSize int64
	Caption  string
}

type MediaItem struct {
	Type     string
	FileID   string
	FileSize int64
}

type MediaDiff struct {
	Added   *MediaItem
	Removed *MediaItem
}

type HandleDeletedPaginationData struct {
	DataID           int64
	ChatID           int64
	Offset           int
	TypeOfPagination string
}

func (h HandleDeletedPaginationData) ToString() string {
	return fmt.Sprintf("%s|%d|%d|%d|%s", consts.CALLBACK_PREFIX_DELETED, h.DataID, h.ChatID, h.Offset, h.TypeOfPagination)
}

type HandleDeletedLogData struct {
	DataID int64
	ChatID int64
	Offset int
}

func (h HandleDeletedLogData) ToString() string {
	return fmt.Sprintf("%s|%d|%d|%d", consts.CALLBACK_PREFIX_DELETED_LOG, h.DataID, h.ChatID, h.Offset)
}

type HandleDeletedMessageData struct {
	MessageID  int
	ChatID     int64
	DataID     int64
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

	return fmt.Sprintf("%s|%d|%d|%d|%d", prefix, h.MessageID, h.ChatID, h.DataID, h.BackOffset)
}

type HandleDeletedFilesDataType int

const (
	HandleDeletedFilesDataTypeMessage HandleDeletedFilesDataType = iota
	HandleDeletedFilesDataTypeData
)

type HandleDeletedFilesData struct {
	MessageID int
	ChatID    int64
	DataID    int64
	Type      HandleDeletedFilesDataType
}

func (h HandleDeletedFilesData) ToString() string {
	switch h.Type {
	case HandleDeletedFilesDataTypeMessage:
		return fmt.Sprintf("%s|%d|%d|%d", consts.CALLBACK_PREFIX_DELETED_FILES, h.MessageID, h.ChatID, HandleDeletedFilesDataTypeMessage)
	case HandleDeletedFilesDataTypeData:
		return fmt.Sprintf("%s|%d|%d|%d", consts.CALLBACK_PREFIX_DELETED_FILES, h.DataID, h.ChatID, HandleDeletedFilesDataTypeData)
	default:
		return ""
	}
}

type HandleEditedDataType int

const (
	HandleEditedDataTypeLog HandleEditedDataType = iota
	HandleEditedDataTypeFiles
)

type HandleEditedData struct {
	DataID int64
	ChatID int64
}

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
	return fmt.Sprintf("%s|%d|%d", prefix, h.ChatID, h.DataID)
}

type HandleBusinessData struct {
	DataID int64
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

	return fmt.Sprintf("%s|%d|%d", prefix, h.DataID, h.ChatID)
}
