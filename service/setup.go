package service

import (
	"encoding/json"
	"errors"
	"github.com/voyage-finance/voyage-tg-server/models"
	"gorm.io/gorm"
	"log"
	"strconv"
	"strings"
)

func (s *Service) SetupChat(id int64, title string, userId int64, userName string) {
	log.Printf("SetupChat id: %d, title: %s, user: %v\n", id, title, userName)
	// create user if not exist
	_ = s.GetOrCreateUser(userId, userName)
	var chat models.Chat
	s.DB.First(&chat, "chat_id = ?", id)
	if !chat.Init {
		log.Println("start creating chat...")
		s.DB.Create(&models.Chat{ChatId: id, Title: title, Init: true})
	}
}

func (s *Service) AddSigner(id int64, name string, address string) string {
	log.Printf("AddSigner id: %d, name: %s, address: %s\n", id, name, address)
	var chat models.Chat
	s.DB.First(&chat, "chat_id = ?", id)
	if !chat.Init {
		return "Please init first"
	}
	address = strings.ToLower(address)
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

func (s *Service) AddSafeWallet(id int64, addr []string) string {
	log.Printf("AddSafeWallet id: %d, address: %s\n", id, addr)
	var chat models.Chat
	s.DB.First(&chat, "chat_id = ?", id)
	if !chat.Init {
		return "Please init first"
	}
	s.DB.Model(&chat).Where("chat_id = ?", id).Update("SafeAddress", addr[1])
	s.DB.Model(&chat).Where("chat_id = ?", id).Update("Chain", addr[0])
	return ""
}

func (s *Service) QueryChat(id int64) *models.Chat {
	var chat models.Chat
	s.DB.First(&chat, "chat_id = ?", id)
	return &chat
}

// Get or Create functions

func (s *Service) GetOrCreateUser(userId int64, userName string) models.User {
	// get or create user
	var user models.User
	err := s.DB.First(&user, "user_id = ?", userId).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		user = models.User{UserId: userId, UserName: userName}
		s.DB.Create(&user)
	}
	return user
}

func (s *Service) GetOrCreateSignMessage(chatId int64, userId int64) models.SignMessage {
	var signMessage models.SignMessage
	err := s.DB.First(&signMessage, "chat_id = ? AND user_id = ?", chatId, userId).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		message := "0x" + s.GenerateMessage(10)
		signMessage = models.SignMessage{UserID: userId, ChatID: chatId, Message: message}
		s.DB.Create(&signMessage)
	}
	return signMessage
}
