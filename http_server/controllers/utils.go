package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func SendBotMessage(msg string, link string, chatId int64) bool {
	botApiKey := os.Getenv("BOT_API_KEY")

	url := fmt.Sprintf("https://api.telegram.org/bot%v/sendMessage", botApiKey)
	startButton := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("✍️ Submit it!", link),
		),
	)
	startButtonJson, _ := json.Marshal(startButton)
	data := map[string]string{
		"text":         msg,
		"chat_id":      fmt.Sprintf("%v", chatId),
		"parse_mode":   "Markdown",
		"reply_markup": string(startButtonJson),
	}

	jsonValue, _ := json.Marshal(data)

	_, err := http.Post(url, "application/json", bytes.NewBuffer(jsonValue))

	if err != nil {
		log.Println(err.Error())
		return false
	}
	return true
}
