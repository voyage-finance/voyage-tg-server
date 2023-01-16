package main

import (
	"encoding/json"
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
			msg.Text = `Commands:
					/this: show current wallet info
					/verify: generate random message to sign
					/submitowner: submit message and signature info to complete verify process
					/submitsafe: submit gnosis safe wallet address
					/queue: check transactions in pending pool
					/balance: check token balances
					/safestatus: check wallet status
					/safequeue: generate link to queue UI
					/safehistory: generate link to history UI
			`
		case "this":
			chatId := update.Message.Chat.ID
			chat := s.QueryChat(chatId)

			// 1. Safe address should be bold
			var e1 tgbotapi.MessageEntity
			e1.Type = "bold"
			e1.Offset = 2
			e1.Length = 13
			msg.Entities = append(msg.Entities, e1)

			// 2. Address should be hypelink
			var e2 tgbotapi.MessageEntity
			e2.Type = "code"
			e2.Offset = 16
			e2.Length = 53
			e2.URL = fmt.Sprintf("https://gnosis-safe.io/app/eth:%s/home", chat.SafeAddress)
			msg.Entities = append(msg.Entities, e2)

			// 3. Link button to gnosis safe wallet
			var safeButton = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL("Link", fmt.Sprintf("https://gnosis-safe.io/app/eth:%s/home", chat.SafeAddress)),
				),
			)

			// 4. Owners should be bold
			var e3 tgbotapi.MessageEntity
			e3.Type = "bold"
			e3.Offset = 68
			e3.Length = 7
			msg.Entities = append(msg.Entities, e3)

			msg.Text = fmt.Sprintln("🔓 Safe address")
			msg.Text += fmt.Sprintf("\neth:%s", strings.ToLower(chat.SafeAddress))
			msg.Text += "\n"
			msg.Text += fmt.Sprintln("\n🔑  Owners")

			var ss []models.Signer
			_ = json.Unmarshal([]byte(chat.Signers), &ss)
			for i, s := range ss {
				msg.Text += fmt.Sprintf("\n%d. @%s - %s", i+1, s.Name, s.Address)
			}

			msg.ReplyMarkup = safeButton
		case "verify":
			s.SetupChat(update.Message.Chat.ID, update.Message.Chat.Title)
			message := "0x" + s.GenerateMessage(10)
			r := fmt.Sprintf("https://telegram-bot-ui-two.vercel.app/sign?message=%s&name=%s", message, update.Message.From.String())
			var safeButton = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL("Link", r),
				),
			)
			msg.ReplyMarkup = safeButton
			msg.Text = "Please verify your account address via Sign-In With Ethereum."
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
			var e tgbotapi.MessageEntity
			e.Type = "bold"
			e.Offset = 2
			e.Length = 16
			msg.Entities = append(msg.Entities, e)
		case "submitowner":
			args := update.Message.CommandArguments()
			info := strings.Split(args, " ")
			if len(info) < 2 {
				msg.Text = "Wrong arguments"
			} else {
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
			}

		case "submitsafe":
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
