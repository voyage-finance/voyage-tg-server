package service

import (
	"errors"
	"fmt"
	"github.com/voyage-finance/voyage-tg-server/models"
	"golang.org/x/exp/slices"
	"gorm.io/gorm"
	"log"
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

func (s *Service) AddSigner(chatId int64, userId int64, address string) string {
	log.Printf("AddSigner chatId: %v, userId: %v, address: %s\n", chatId, userId, address)
	var signer models.Signer
	address = strings.ToLower(address)
	forceSave := false

	// 1.0 Find whether user is owner or not
	owners := s.Status(chatId) // lowered in slice
	isSigner := slices.Contains(owners, address)

	var user models.User
	err := s.DB.First(&user, userId).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return err.Error()
	}

	err = s.DB.First(&signer, "user_user_id = ? AND chat_chat_id = ?", userId, chatId).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		forceSave = true
		signer = models.Signer{ChatID: chatId, UserID: userId, Name: user.UserName, Address: address, IsSigner: isSigner}
	}

	// 1.1 check whether address is already allocated to other user in the chat
	var checkSigner models.Signer
	err = s.DB.First(&checkSigner, "chat_chat_id = ? AND user_user_id != ? and address = ?", chatId, userId, address).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf(signer.Address + " is already bind to " + checkSigner.Name)
		return fmt.Sprintf("%v is already bind to other user", checkSigner.Address)
	}

	if signer.ID != 0 {
		// 2.0 Signer already exists in db
		if signer.Address == address {
			// 2.0.1 if address already allocated for given user, do nothing
			if signer.IsSigner != isSigner {
				// 2.0.2 update if isSigner changed
				signer.IsSigner = isSigner
				forceSave = true
				log.Printf("+ %v has different ownership %v", user.UserName, isSigner)
			} else {
				log.Printf("- %v does not changed!", user.UserName)
			}
		} else {
			// 2.1 User changed Address to new
			signer.Address = address
			signer.IsSigner = isSigner
			forceSave = true
		}
	}
	if forceSave {
		s.DB.Save(&signer)
		log.Printf("+/- Signer %v was UPDATED!", signer.ID)
	}
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

func (s *Service) GetOrCreateSignMessage(chatId int64, userId int64, forceUpdate bool) models.SignMessage {
	var signMessage models.SignMessage
	err := s.DB.First(&signMessage, "chat_id = ? AND user_id = ?", chatId, userId).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		message := "0x" + s.GenerateMessage(10)
		signMessage = models.SignMessage{UserID: userId, ChatID: chatId, Message: message}
		s.DB.Create(&signMessage)
	}
	//else if forceUpdate {
	//	signMessage.Message = "0x" + s.GenerateMessage(10)
	//	//signMessage.IsVerified = false
	//	s.DB.Save(&signMessage)
	//}
	return signMessage
}

func (s *Service) RemoveSigner(signMessage models.SignMessage) string {
	// 1.0 remove Signer from DB
	var signer models.Signer
	safeAddress := signer.Address
	s.DB.Where("chat_chat_id = ? AND user_user_id = ?", signMessage.ChatID, signMessage.UserID).Delete(&signer)
	log.Printf("Signers=%v was deleted!", safeAddress)

	// 2.0 remove signMessage
	s.DB.Delete(&signMessage)
	return fmt.Sprintf("Address( %v ) was removed!", safeAddress)
}

func (s *Service) GetSignersByChat(chat *models.Chat) []models.Signer {
	var signers []models.Signer
	s.DB.Find(&signers, "chat_chat_id = ?", chat.ChatId)
	return signers
}
