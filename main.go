package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/go-resty/resty/v2"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/voyage-finance/voyage-tg-server/models"
	"github.com/voyage-finance/voyage-tg-server/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema
	db.AutoMigrate(&models.User{})
	db.AutoMigrate(&models.Chat{})

	client := resty.New()

	s := service.Service{DB: db, Client: client}

	bot, err := tgbotapi.NewBotAPI("5835886666:AAGt66BQaepE3VAGACDvGSmk2qFFUqo2fEY")
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { // ignore non-Message updates
			continue
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)

		// Extract the command from the Message.
		switch update.Message.Command() {
		case "help":
			msg.Text = "I understand /this, /setup, /initiate, /sign and /execute"
		case "this":
			chatId := update.Message.Chat.ID
			chat := s.QueryChat(chatId)
			sender := update.Message.From.String()
			msg.Text = fmt.Sprintf(`Channel info:
					Chat ID: %d
					Init: %t
					Sender: %s
					Title: %s
					Safe Address: https://gnosis-safe.io/app/eth:%s/home 
					Signer: %s
			`, chatId, chat.Init, sender, chat.Title, chat.SafeAddress, chat.Signers)
		case "verify":
			s.SetupChat(update.Message.Chat.ID, update.Message.Chat.Title)
			message := "0x" + s.GenerateMessage(10)
			r := fmt.Sprintf("\nhttps://telegram-bot-ui-two.vercel.app/sign?message=%s&name=%s", message, update.Message.From.String())
			msg.Text = "To verify your address please sign message: \n" + r
			msg.Text += "\n\nAfterwards, please submit the signatue as following format: submitowner message signature"
		case "queue":
			args := update.Message.CommandArguments()
			limit, err := strconv.ParseInt(args, 10, 64)
			if err != nil {
				msg.Text = "Wrong argument"
			} else {
				msg.Text = s.QueueTransaction(update.Message.Chat.ID, limit)
			}
		case "balance":
			chatId := update.Message.Chat.ID
			msg.Text = s.QueryTokenBalance(chatId)
		case "initiate":
			msg.Text = "Command initiate"
		case "sign":
			msg.Text = "Command sign"
		case "execute":
			msg.Text = "Command execute"
		case "submitowner":
			args := update.Message.CommandArguments()
			info := strings.Split(args, " ")
			message, err := hexutil.Decode(info[1])
			if err != nil {
				msg.Text = "Wrong message"
			}
			signature, err := hexutil.Decode(info[2])
			if err != nil {
				msg.Text = "Wrong signature"
			}
			addr := s.RecoveryAddress(message, signature)
			ret := s.AddSigner(update.Message.Chat.ID, info[0], addr)
			if ret != "" {
				msg.Text = ret
			} else {
				msg.Text = fmt.Sprintf("Added signer, address: %s", addr)
			}
		case "safesubmit":
			args := update.Message.CommandArguments()
			ret := s.AddSafeWallet(update.Message.Chat.ID, args)
			if ret != "" {
				msg.Text = ret
			} else {
				msg.Text = fmt.Sprintf("Added safe wallet, address: %s", args)
			}

		case "safestatus":
			msg.Text = s.Status(update.Message.Chat.ID)
		case "safequeue":
			msg.Text = s.GenerateQueueLink(update.Message.Chat.ID)
		case "safehistory":
			msg.Text = s.GenerateHistoryLink(update.Message.Chat.ID)
		default:
			msg.Text = "I don't know that command"
		}

		if _, err := bot.Send(msg); err != nil {
			log.Panic(err)
		}
	}
}
