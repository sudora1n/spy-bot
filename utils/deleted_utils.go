package utils

import (
	"ssuspy-bot/consts"
	"ssuspy-bot/repository"
	"ssuspy-bot/types"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func DeletedRows(
	chatID int64,
	user *repository.User,
	loc *i18n.Localizer,
	oldMsgs []*telego.Message,
	pagination *repository.PaginationAnswer,
	offset int,
	dataID int64,
) (rows [][]telego.InlineKeyboardButton) {
	if len(oldMsgs) > 0 {
		data := types.HandleDeletedLogData{
			DataID: dataID,
			ChatID: chatID,
		}
		rows = append(rows,
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton(
					loc.MustLocalize(&i18n.LocalizeConfig{
						MessageID: "business.deleted.fullMessages",
					}),
				).WithCallbackData(data.ToString()),
			),
		)

		filesLen := 0
		for _, msg := range oldMsgs {
			media := GetFile(msg)
			if media != nil {
				filesLen++
			}
		}

		if filesLen != 0 {
			callbackData := types.HandleDeletedFilesData{
				DataID: dataID,
				ChatID: chatID,
				Type:   types.HandleDeletedFilesDataTypeData,
			}
			rows = append(rows,
				tu.InlineKeyboardRow(
					tu.InlineKeyboardButton(
						loc.MustLocalize(&i18n.LocalizeConfig{
							MessageID: "business.deleted.request.files",
							TemplateData: map[string]int{
								"Count": filesLen,
							},
							PluralCount: filesLen,
						}),
					).WithCallbackData(callbackData.ToString()),
				),
			)
		}

		var newMsgs []*telego.Message
		if len(oldMsgs) > consts.MAX_BUTTONS {
			newMsgs = oldMsgs[:consts.MAX_BUTTONS]
		} else {
			newMsgs = oldMsgs
		}

		for i := 0; i < len(newMsgs); i += 2 {
			row := make([]telego.InlineKeyboardButton, 0, 2)
			data := types.HandleDeletedMessageData{
				MessageID: newMsgs[i].MessageID,
				ChatID:    chatID,
				DataID:    dataID,
			}

			row = append(row, tu.InlineKeyboardButton(
				loc.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "business.deleted.messageItem",
					TemplateData: map[string]int{
						"Count": i + 1 + offset,
					},
				}),
			).WithCallbackData(data.ToString(types.HandleDeletedMessageDataTypeMessage)))

			if i+1 < len(newMsgs) {
				data.MessageID = newMsgs[i+1].MessageID
				row = append(row, tu.InlineKeyboardButton(
					loc.MustLocalize(&i18n.LocalizeConfig{
						MessageID: "business.deleted.messageItem",
						TemplateData: map[string]int{
							"Count": i + 2 + offset,
						},
					}),
				).WithCallbackData(data.ToString(types.HandleDeletedMessageDataTypeMessage)))
			}

			rows = append(rows, row)
		}

		if len(oldMsgs) > consts.MAX_BUTTONS || pagination.Backward || pagination.Forward {
			row := make([]telego.InlineKeyboardButton, 0, 2)

			paginationData := types.HandleDeletedPaginationData{
				DataID: dataID,
				ChatID: chatID,
				Offset: offset,
			}

			if pagination.Backward {
				paginationData.TypeOfPagination = "b"
				row = append(
					row,
					tu.InlineKeyboardButton(
						loc.MustLocalize(&i18n.LocalizeConfig{
							MessageID: "arrow.backward",
						}),
					).
						WithCallbackData(paginationData.ToString()),
				)
			}
			if pagination.Forward {
				paginationData.TypeOfPagination = "f"
				row = append(
					row,
					tu.InlineKeyboardButton(
						loc.MustLocalize(&i18n.LocalizeConfig{
							MessageID: "arrow.forward",
						}),
					).
						WithCallbackData(paginationData.ToString()),
				)
			}

			rows = append(rows, row)
		}
	}

	return rows
}
