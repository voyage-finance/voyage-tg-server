package controllers

import (
	"encoding/json"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"net/http"
	"os"
)

func ReturnHttpBadResponse(rw http.ResponseWriter, response string) {
	rw.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(rw).Encode(response)
	log.Println(response)
}

type ServerBot struct {
	bot *tgbotapi.BotAPI
}

func NewServerBot() *ServerBot {
	botApiKey := os.Getenv("BOT_API_KEY")

	bot, err := tgbotapi.NewBotAPI(botApiKey)
	if err != nil {
		log.Panic(err)
		return nil
	}
	return &ServerBot{bot}
}

func (serverBot *ServerBot) SendBotMessage(msg string, link string, chatId int64) bool {
	startButton := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("✍️ Submit it!", link),
		),
	)
	message := tgbotapi.NewMessage(chatId, msg)
	message.ParseMode = "Markdown"
	message.ReplyMarkup = startButton
	if _, err := serverBot.bot.Send(message); err != nil {
		log.Println(err)
		return false
	}

	return true
}
