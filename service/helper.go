package service

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func GetNavigationInstruction(bot *tgbotapi.BotAPI, s Service, chatId int64, msg *tgbotapi.MessageConfig) {
	chat := s.QueryChat(chatId)
	if chat.SafeAddress == "" {
		msg.Text = "Please /setup Safe address in the chat (*only admin is allowed*)"
		msg.ReplyMarkup = GetSingleSetupButton()
		return
	} else {
		msg.Text = "Please /link your wallet address to Telegram account in the chat"
		msg.ReplyMarkup = GetSingleLinkButton()
	}
}
