package consts

const MAX_FILE_SIZE_BYTES = 20_000_000
const MAX_FILE_SIZE_BYTES_LOCAL = 2_000_000_000
const MAX_MEDIA_GROUP_SIZE = 10

const MAX_LEN = 4096
const MAX_BUTTONS = 8

const MAX_NAME_LEN = 128
const MAX_MESSAGE_TEXT_LEN = 256

const DATETIME_FOR_FILES = "02-01-2006_15-04-05"
const DATETIME_FOR_MESSAGE = "02-01-2006 15:04:05"

// максимальная длинна callback data - 64 байта
// 1 символ - 1 байт, так что учитываем эту длинну при создании callback data
// поэтому были введены mongodb data таблицы, т.к. иногда мы не можем передать все необходимые сообщения
// в callback data, их может придти до 100 сообщений в deleted business message хендлере
// int64 в utf8 имеет длинну 19 байт, datetime обычно до 11 включительно
const CALLBACK_PREFIX_DELETED = "---1"

const CALLBACK_PREFIX_DELETED_LOG = "___1"
const CALLBACK_PREFIX_DELETED_PAGINATION = "___2"
const CALLBACK_PREFIX_DELETED_MESSAGE = "___3"
const CALLBACK_PREFIX_DELETED_DETAILS = "___4"
const CALLBACK_PREFIX_DELETED_FILES = "___5"

const CALLBACK_PREFIX_EDITED_LOG = "___6"
const CALLBACK_PREFIX_EDITED_FILES = "___7"

const CALLBACK_PREFIX_LANG = "___8"
const CALLBACK_PREFIX_LANG_CHANGE = "___9"

const CALLBACK_PREFIX_BACK_TO_START = "__10"

const REDIS_QUEUE_FILES = "queue:files"
const REDIS_RATELIMIT_COUNT = "rl_count"
const REDIS_RATELIMIT_QUEUE = "rl_queue"
const REDIS_RATELIMIT_QUEUE_BUSINESS = "rl_queue_business"
const REDIS_RATELIMIT_QUEUE_BUSINESS_CONNECTION = "rl_queue_business_connection"

const MonthInSeconds = 30 * 24 * 60 * 60
