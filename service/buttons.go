package service

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

func GetHelperButtons() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("đ Link", "/link"),
			tgbotapi.NewInlineKeyboardButtonData("âšī¸ Help", "/help"),
		),
	)
}

func GetSingleSetupButton() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("âī¸ Setup", "/setup"),
		),
	)
}

func GetSingleLinkButton() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("đ Link", "/link"),
		),
	)
}
