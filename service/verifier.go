package service

import (
	"crypto/sha256"
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/voyage-finance/voyage-tg-server/models"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-resty/resty/v2"
	"gorm.io/gorm"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

type TokenInfo struct {
	TokenAddress string
	TokenType    string
	Name         string
	Symbol       string
	Decimals     int64
}

type Service struct {
	DB        *gorm.DB
	Client    *resty.Client
	EthClient *ethclient.Client
	Tokens    map[string]TokenInfo
}

func (s *Service) UpdatePoints(chatId int64, addr string, point int64) {
	addr = strings.ToLower(addr)
	var signer models.Signer
	err := s.DB.First(&signer, "chat_chat_id = ? AND address = ?", chatId, addr).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return
	}
	signer.Points += point
	s.DB.Save(&signer)
}

func (s *Service) UpdateLastConfirmedNonce(id int64, nonce int64) {
	chat := s.QueryChat(id)
	s.DB.Model(&chat).Where("chat_id = ?", id).Update("LastConfirmNonce", nonce)
}

func (s *Service) GetCurrentLastConfirmedNonce(id int64) int64 {
	chat := s.QueryChat(id)
	return chat.LastConfirmNonce
}

func (s *Service) GenerateMessage(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	t := time.Now().String()
	b = append(b, []byte(t)...)
	r := sha256.Sum256(b)
	return fmt.Sprintf("%x", r)
}

func (s *Service) RecoveryAddress(message []byte, signature []byte) string {
	sig := signature

	message = accounts.TextHash(message)
	if sig[crypto.RecoveryIDOffset] == 27 || sig[crypto.RecoveryIDOffset] == 28 {
		sig[crypto.RecoveryIDOffset] -= 27 // Transform yellow paper V from 27/28 to 0/1
	}

	recovered, _ := crypto.SigToPub(message, sig)

	recoveredAddr := crypto.PubkeyToAddress(*recovered)
	return recoveredAddr.Hex()
}

func (s *Service) SendVerifyButton(bot *tgbotapi.BotAPI, update tgbotapi.Update, signMessage models.SignMessage) {
	r := fmt.Sprintf("%v/sign?message=%s&name=%s&msg_id=%v", os.Getenv("FRONT_URL"), signMessage.Message, update.Message.From.String(), signMessage.ID)
	var safeButton = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("Link", r),
		),
	)

	var chat models.Chat
	s.DB.First(&chat, "chat_id = ?", signMessage.ChatID)
	if !chat.Init {
		log.Println("SendVerifyButton error: chat does not exist")
		return
	}

	dmText := "Please verify, your account address via Sign-In With Ethereum"
	if signMessage.IsVerified {
		//dmText += "You have already verified the account address via Sign-In With Ethereum"
		dmText += fmt.Sprintf(". Your current bind address=`%v`", s.GetAddressByUsername(&chat, update.Message.From.String()))
	}

	dmText += fmt.Sprintf(". Chat: `%v`", chat.Title)
	dmMsg := tgbotapi.NewMessage(update.Message.From.ID, dmText)
	dmMsg.ParseMode = "Markdown"
	dmMsg.ReplyMarkup = safeButton
	if _, err := bot.Send(dmMsg); err != nil {
		log.Println(err)
	}
}

func (s *Service) SendLinkButton(bot *tgbotapi.BotAPI, update tgbotapi.Update, signMessage models.SignMessage) {
	// here we can validate whether user is admin or not
	r := fmt.Sprintf("%v/link?message=%s&name=%s&msg_id=%v", os.Getenv("FRONT_URL"), signMessage.Message, update.Message.From.String(), signMessage.ID)
	var safeButton = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("Choose Safe", r),
		),
	)

	var chat models.Chat
	s.DB.First(&chat, "chat_id = ?", signMessage.ChatID)
	if !chat.Init {
		log.Println("SendVerifyButton error: chat does not exist")
		return
	}

	dmText := fmt.Sprintf("Please sign message via Sign-In With Ethereum and choose your Safe account to link in chat `%v`", chat.Title)
	if chat.SafeAddress != "" {
		dmText = fmt.Sprintf("Chat %v is already linked to Safe `%v`.\nPress the link button below if you wish to update Safe", chat.Title, chat.SafeAddress)
	}

	dmMsg := tgbotapi.NewMessage(update.Message.From.ID, dmText)
	dmMsg.ReplyMarkup = safeButton
	dmMsg.ParseMode = "Markdown"
	if _, err := bot.Send(dmMsg); err != nil {
		log.Println(err)
	}

}
