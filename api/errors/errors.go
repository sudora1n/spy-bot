package errors

import (
	"errors"

	"connectrpc.com/connect"
)

var (
	ERROR_INTERNAL           = errors.New("internal error")
	ERROR_TOO_MANY_BOTS      = errors.New("too many bots")
	ERROR_ALREADY_EXISTS     = errors.New("bot already exists")
	ERROR_INVALID_PARAM      = errors.New("invalid param")
	ERROR_INVALID_TOKEN      = errors.New("invalid bot token")
	ERROR_INCORRECT_SETTINGS = errors.New("incorrect settings in @botfather")
	ERROR_PERMISSION_DENIED  = errors.New("it's not your bot ^^(")
	ERROR_USER_NOT_FOUND     = errors.New("user not found")
)

var (
	CONNECT_ERROR_INTERNAL = connect.NewError(connect.CodeInternal, ERROR_INTERNAL)
	CONNECT_USER_NOT_FOUND = connect.NewError(connect.CodeNotFound, ERROR_USER_NOT_FOUND)
)
