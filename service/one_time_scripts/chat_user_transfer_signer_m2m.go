package one_time_scripts

import (
	"encoding/json"
	"github.com/voyage-finance/voyage-tg-server/models"
	"github.com/voyage-finance/voyage-tg-server/service"
	"log"
)

func SaveAllSignersInChat(s service.Service, chat models.Chat) {
	var signers []models.Signer
	if chat.Signers != "" {
		err := json.Unmarshal([]byte(chat.Signers), &signers)
		if err != nil {
			log.Printf("Parsing Singer is failed: %s\n", err.Error())
			return
		}
	}
	for _, signer := range signers {
		signer.ChatID = chat.ChatId
		var user models.User
		s.DB.First(&user, "user_name = ?", signer.Name)
		signer.UserID = user.UserId
		s.DB.Create(&signer)
	}
}

func TransferSignersToTableInAllChats(s service.Service) {
	var chats []models.Chat
	s.DB.Find(&chats)
	for _, chat := range chats {
		SaveAllSignersInChat(s, chat)
	}
}
