package one_time_scripts

import (
	"github.com/voyage-finance/voyage-tg-server/models"
	"github.com/voyage-finance/voyage-tg-server/service"
)

func UpdateChatSignersOwnership(s service.Service, chat models.Chat) {
	var signers []models.Signer
	s.DB.Find(&signers, "chat_chat_id = ?", chat.ChatId)
	for _, signer := range signers {
		s.AddSigner(chat.ChatId, signer.UserID, signer.Address)
	}
}

func UpdateAllSignersOwnership(s service.Service) {
	var chats []models.Chat
	s.DB.Find(&chats)
	for _, chat := range chats {
		UpdateChatSignersOwnership(s, chat)
	}
}
