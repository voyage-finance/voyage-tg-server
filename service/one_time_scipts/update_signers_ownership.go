package one_time_scipts

import (
	"encoding/json"
	"github.com/voyage-finance/voyage-tg-server/models"
	"github.com/voyage-finance/voyage-tg-server/service"
	"log"
)

func UpdateChatSignersOwnership(s service.Service, chat models.Chat) {
	var signers []models.Signer
	if chat.Signers != "" {
		err := json.Unmarshal([]byte(chat.Signers), &signers)
		if err != nil {
			log.Printf("Parsing Singer is failed: %s\n", err.Error())
			return
		}
	}
	for _, signer := range signers {
		s.AddSigner(chat.ChatId, signer.Name, signer.Address)
	}
}

func UpdateAllSignersOwnership(s service.Service) {
	var chats []models.Chat
	s.DB.Find(&chats)
	for _, chat := range chats {
		UpdateChatSignersOwnership(s, chat)
	}
}
