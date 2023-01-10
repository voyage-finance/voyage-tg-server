package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
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

	s := service.Service{DB: db}

	bot, err := tgbotapi.NewBotAPI("5830732458:AAHtcj5oGrX8cbqjXiX_wNtS8tJXQVZojoo")
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
		fmt.Println("chat id: ", update.Message.Chat.ID)
		fmt.Println("message: ", update.Message.Text)

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
		case "setup":
			s.SetupChat(update.Message.Chat.ID, update.Message.Chat.Title)
			r := s.GenerateMessage(10)
			msg.Text = "Please sign message: " + r
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
		case "submitsafe":
			args := update.Message.CommandArguments()
			ret := s.AddSafeWallet(update.Message.Chat.ID, args)
			if ret != "" {
				msg.Text = ret
			} else {
				msg.Text = fmt.Sprintf("Added safe wallet, address: %s", args)
			}
		default:
			msg.Text = "I don't know that command"
		}

		if _, err := bot.Send(msg); err != nil {
			log.Panic(err)
		}
	}
}
