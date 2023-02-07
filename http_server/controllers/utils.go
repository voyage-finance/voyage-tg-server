package controllers

import (
	"encoding/json"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/voyage-finance/voyage-tg-server/service"
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

func ConstructRequestMessage(msg string, link string, chatId int64) tgbotapi.MessageConfig {
	startButton := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("✍️ Submit it!", link),
		),
	)
	message := tgbotapi.NewMessage(chatId, msg)
	message.ParseMode = "Markdown"
	message.ReplyMarkup = startButton
	return message
}

func ConstructSignupMessage(msg string, chatId int64) tgbotapi.MessageConfig {
	button := service.GetHelperButtons()
	message := tgbotapi.NewMessage(chatId, msg)
	message.ParseMode = "Markdown"
	message.ReplyMarkup = button
	return message
}

func (serverBot *ServerBot) SendBotMessage(message tgbotapi.MessageConfig) bool {
	if _, err := serverBot.bot.Send(message); err != nil {
		log.Printf("SendBotMessage error: %s\n", err.Error())
		return false
	}

	return true
}
