package service

import (
	"encoding/json"
	"github.com/voyage-finance/voyage-tg-server/models"
	"log"
	"strconv"
)

func (s *Service) SetupChat(id int64, title string) {
	log.Printf("SetupChat id: %d, title: %s\n", id, title)
	var chat models.Chat
	s.DB.First(&chat, "chat_id = ?", id)
	if !chat.Init {
		log.Println("start creating chat...")
		s.DB.Create(&models.Chat{ChatId: id, Title: title, Init: true})
	}
}

func (s *Service) AddPendingVerification(message string, chatId string, name string, address string) {

}

func (s *Service) AddSigner(id int64, name string, address string) string {
	log.Printf("AddSigner id: %d, name: %s, address: %s\n", id, name, address)
	var chat models.Chat
	s.DB.First(&chat, "chat_id = ?", id)
	if !chat.Init {
		return "Please init first"
	}
	var signers []models.Signer
	if chat.Signers != "" {
		err := json.Unmarshal([]byte(chat.Signers), &signers)
		if err != nil {
			log.Printf("AddSigner failed, error: %s\n", err.Error())
			return "Get current signer failed"
		}
	}

	for _, s := range signers {
		if s.Address == address {
			return ""
		}
	}
	signers = append(signers, models.Signer{Name: name, Address: address})
	signerStr, err := json.Marshal(signers)
	if err != nil {
		return "Marshal signers faled"
	}
	s.DB.Model(&chat).Where("chat_id = ?", strconv.FormatInt(id, 10)).Update("Signers", signerStr)
	return ""
}

func (s *Service) AddSafeWallet(id int64, addr string) string {
	log.Printf("AddSafeWallet id: %d, address: %s\n", strconv.FormatInt(id, 10), addr)
	if addr == "" {
		return "Wrong address"
	}
	var chat models.Chat
	s.DB.First(&chat, "chat_id = ?", id)
	if !chat.Init {
		return "Please init first"
	}
	s.DB.Model(&chat).Where("chat_id = ?", id).Update("SafeAddress", addr)
	return ""
}

func (s *Service) QueryChat(id int64) *models.Chat {
	var chat models.Chat
	s.DB.First(&chat, "chat_id = ?", id)
	return &chat
}
