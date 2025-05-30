package handlers

import (
	"ssuspy-bot/repository"
	"ssuspy-bot/utils"
	"strings"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/rs/zerolog"
)

func HandleInlineQuery(c *th.Context, update telego.Update) error {
	query := update.InlineQuery
	loc := c.Value("loc").(*i18n.Localizer)
	iUser := c.Value("iUser").(*repository.IUser)

	button := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(
				loc.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "inlineQuery.handleUserGiftUpgrade.button.text",
				}),
			).WithCopyText(&telego.CopyTextButton{
				Text: loc.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "inlineQuery.handleUserGiftUpgrade.button.buttonCopy",
				}),
			}),
		),
	)

	var result *telego.AnswerInlineQueryParams
	connection := iUser.BotUser.GetUserCurrentConnection()
	if connection == nil {
		result = tu.InlineQuery(
			query.ID,
			tu.ResultArticle(
				"needBusiness",
				loc.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "inlineQuery.needBusiness",
				}),
				tu.TextMessage(loc.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "inlineQuery.needBusiness",
				})),
			),
		)
	} else {
		result = tu.InlineQuery(
			query.ID,
			tu.ResultArticle(
				"userGiftUpgrade",
				loc.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "inlineQuery.handleUserGiftUpgrade.text",
				}),
				tu.TextMessage(
					loc.MustLocalize(&i18n.LocalizeConfig{
						MessageID: "inlineQuery.handleUserGiftUpgrade.textMessage",
					}),
				),
			).WithReplyMarkup(button),
		)
	}

	return c.Bot().AnswerInlineQuery(c, result.WithIsPersonal().WithCacheTime(0))
}

func HandleUserGiftUpgrade(c *th.Context, update telego.Update) error {
	query := update.ChosenInlineResult
	log := c.Value("log").(*zerolog.Logger)
	loc := c.Value("loc").(*i18n.Localizer)
	iUser := c.Value("iUser").(*repository.IUser)

	connection := iUser.BotUser.GetUserCurrentConnection()
	rights, err := utils.GetBusinessRights(c, connection)
	if err != nil {
		log.Warn().Err(err).Msg("failed get business connection")
		return err
	}

	log.Debug().Any("rights", rights).Msg("rights info")
	if !rights.CanTransferAndUpgradeGifts || !rights.CanViewGiftsAndStars {
		text := ""
		switch {
		case !rights.CanTransferAndUpgradeGifts:
			text = "errors.userHandlers.noCanTransferAndUpgradeGifts"
		case !rights.CanViewGiftsAndStars:
			text = "errors.userHandlers.noCanViewGiftsAndStars"
		}
		_, err := c.Bot().EditMessageText(
			c,
			&telego.EditMessageTextParams{
				InlineMessageID: query.InlineMessageID,
				Text: loc.MustLocalize(&i18n.LocalizeConfig{
					MessageID: text,
				}),
				ParseMode:   telego.ModeHTML,
				ReplyMarkup: nil,
			},
		)
		return err
	}

	_, err = c.Bot().EditMessageText(
		c,
		&telego.EditMessageTextParams{
			InlineMessageID: query.InlineMessageID,
			Text: loc.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "userCallbackHandlers.handleUserGiftUpgrade.part1",
			}),
			ParseMode: telego.ModeHTML,
		},
	)
	if err != nil {
		log.Warn().Err(err).Msg("error while editing message")
	}

	gifts, err := c.Bot().GetBusinessAccountGifts(c, &telego.GetBusinessAccountGiftsParams{
		BusinessConnectionID: connection.ID,
		ExcludeUnlimited:     true,
		ExcludeUnique:        true,
	})
	if err != nil {
		log.Warn().Err(err).Msg("failed get business gifts")
		return err
	}

	userBalance, err := c.Bot().GetBusinessAccountStarBalance(c, &telego.GetBusinessAccountStarBalanceParams{
		BusinessConnectionID: connection.ID,
	})
	if err != nil {
		log.Error().Err(err).Msg("error getting user balance")
		return err
	}
	balance := userBalance.Amount
	log.Debug().Int("balance", balance).Int("nanoBalance", userBalance.NanostarAmount).Msg("user balance info")

	var upgradeGifts []*telego.OwnedGiftRegular
	for _, untypedGift := range gifts.Gifts {
		if gift, ok := untypedGift.(*telego.OwnedGiftRegular); ok {
			log.Debug().Str("giftID", gift.Gift.ID).Msg("gift info")
			if !gift.CanBeUpgraded {
				log.Debug().Msg("skipped due cant be upgraded")
				continue
			}

			if gift.Gift.UpgradeStarCount > balance {
				log.Debug().Int("balance", balance).Int("cost", gift.Gift.UpgradeStarCount).Msg("skipped due price to convert bigger than user balance")
				continue
			}

			upgradeGifts = append(upgradeGifts, gift)
		}
	}
	if len(upgradeGifts) == 0 {
		_, err = c.Bot().EditMessageText(
			c,
			&telego.EditMessageTextParams{
				InlineMessageID: query.InlineMessageID,
				Text: loc.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "userCallbackHandlers.handleUserGiftUpgrade.noFound",
				}),
				ParseMode:   telego.ModeHTML,
				ReplyMarkup: nil,
			},
		)
		return err
	}

	counter := 0
	giftsIDs := make([]string, 0, 5)
	stringGiftsIDs := make([]string, 0, 5)
	for _, gift := range upgradeGifts {
		amount := gift.Gift.UpgradeStarCount
		if amount > balance {
			continue
		}

		err = c.Bot().UpgradeGift(c, &telego.UpgradeGiftParams{
			BusinessConnectionID: connection.ID,
			OwnedGiftID:          gift.OwnedGiftID,
			KeepOriginalDetails:  true,
			StarCount:            amount,
		})
		if err != nil {
			log.Warn().Err(err).Msg("failed update gift")
			continue
		}
		balance -= amount
		counter++
		giftsIDs = append(giftsIDs, gift.Gift.ID)

		if counter >= 5 {
			joinedGifts := strings.Join(giftsIDs, ", ")
			if len(stringGiftsIDs) >= 5 {
				stringGiftsIDs = stringGiftsIDs[1:]
			}
			stringGiftsIDs = append(stringGiftsIDs, joinedGifts)

			_, err = c.Bot().EditMessageText(
				c,
				&telego.EditMessageTextParams{
					InlineMessageID: query.InlineMessageID,
					Text: loc.MustLocalize(&i18n.LocalizeConfig{
						MessageID: "userCallbackHandlers.handleUserGiftUpgrade.part2",
						TemplateData: map[string]string{
							"Gifts": strings.Join(stringGiftsIDs, "\n\n"),
						},
					}),
					ParseMode: telego.ModeHTML,
				},
			)

			if err != nil {
				log.Warn().Err(err).Msg("error while editing upgrade gift message")
			}

			counter = 0
			giftsIDs = giftsIDs[:0]
		}
	}

	if counter > 0 {
		_, err = c.Bot().EditMessageText(
			c,
			&telego.EditMessageTextParams{
				InlineMessageID: query.InlineMessageID,
				Text: loc.MustLocalize(&i18n.LocalizeConfig{
					MessageID: "userCallbackHandlers.handleUserGiftUpgrade.final",
					TemplateData: map[string]string{
						"Gifts": strings.Join(stringGiftsIDs, "\n\n"),
					},
				}),
				ParseMode:   telego.ModeHTML,
				ReplyMarkup: nil,
			},
		)
		if err != nil {
			log.Warn().Err(err).Msg("error while editing final message")
		}
	}

	return err
}
