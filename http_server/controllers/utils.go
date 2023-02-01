package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

func ReturnHttpBadResponse(rw http.ResponseWriter, response string) {
	rw.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(rw).Encode(response)
	log.Println(response)
}

func SendBotMessage(msg string, chatId int64) bool {
	botApiKey := os.Getenv("BOT_API_KEY")

	url := fmt.Sprintf("https://api.telegram.org/bot%v/sendMessage", botApiKey)
	data := map[string]string{
		"text":       msg,
		"chat_id":    fmt.Sprintf("%v", chatId),
		"parse_mode": "Markdown",
	}

	jsonValue, _ := json.Marshal(data)

	_, err := http.Post(url, "application/json", bytes.NewBuffer(jsonValue))

	if err != nil {
		log.Println(err.Error())
		return false
	}
	return true
}
