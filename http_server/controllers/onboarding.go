package controllers

// tutorial: https://dev.to/hackmamba/build-a-rest-api-with-golang-and-mongodb-gorillamux-version-57fh

import (
	"encoding/json"
	"fmt"
	"github.com/voyage-finance/voyage-tg-server/models"
	"gorm.io/gorm"
	"net/http"
)

func Test(db *gorm.DB) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var chats []models.Chat
		db.Find(&chats)
		fmt.Println(chats, "------")
		json.NewEncoder(rw).Encode(chats)
	}
}
