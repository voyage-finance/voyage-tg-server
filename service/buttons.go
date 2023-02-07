package service

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

func GetHelperButtons() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔗 Link", "/link"),
			tgbotapi.NewInlineKeyboardButtonData("ℹ️ Help", "/help"),
		),
	)
}

func GetSingleSetupButton() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Setup", "/setup"),
		),
	)
}

func GetSingleLinkButton() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔗 Link", "/link"),
		),
	)
}
