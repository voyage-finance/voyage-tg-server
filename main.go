package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
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

	tokens, err := os.ReadFile("tokens.json")
	if err != nil {
		panic("failed to read tokens")
	}
	var ts []service.TokenInfo
	json.Unmarshal(tokens, &ts)

	client := resty.New()
	infuraAPI := "https://mainnet.infura.io/v3/b85b517bc9ed49bd8fe5eacb6d9a2bc7"
	ethClient, err := ethclient.Dial(infuraAPI)
	if err != nil {
		log.Fatal(err)
	}

	tokenInfo := make(map[string]service.TokenInfo)
	for _, t := range ts {
		tokenInfo[strings.ToLower(t.TokenAddress)] = t
	}

	s := service.Service{DB: db, Client: client, EthClient: ethClient, Tokens: tokenInfo}

	bot, err := tgbotapi.NewBotAPI("5835886666:AAGt66BQaepE3VAGACDvGSmk2qFFUqo2fEY")
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	// n := service.Notification{Bot: bot, S: &s}
	// go n.Start()

	for update := range updates {
		if update.Message == nil { // ignore non-Message updates
			continue
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)

		// Extract the command from the Message.
		switch update.Message.Command() {
		case "help":
			log.Printf("Chat id: %d\n", update.Message.Chat.ID)
			msg.Text = `Commands:
					/this: show current wallet info
					/verify: generate random message to sign
					/submitowner: submit message and signature info to complete verify process
					/setup: submit gnosis safe wallet address
					/queue: check transactions in pending pool
					/balance: check token balances
					/safestatus: check wallet status
					/safequeue: generate link to queue UI
					/safehistory: generate link to history UI
			`
		case "this":
			chatId := update.Message.Chat.ID
			chat := s.QueryChat(chatId)

			s1 := "🔓 Safe address\n"
			msg.Text = s1
			var e1 tgbotapi.MessageEntity
			e1.Type = "bold"
			e1.Offset = 0
			e1.Length = len(utf16.Encode([]rune(s1)))
			msg.Entities = append(msg.Entities, e1)

			addr := common.HexToAddress(chat.SafeAddress)
			s2 := fmt.Sprintf("\n%s:%s\n", chat.Chain, addr.Hex())
			var e2 tgbotapi.MessageEntity
			e2.Type = "code"
			e2.Offset = len(utf16.Encode([]rune(s1)))
			e2.Length = len(utf16.Encode([]rune(s2)))
			e2.URL = fmt.Sprintf("https://app.safe.global/%s:%s/home", chat.Chain, addr.Hex())
			msg.Entities = append(msg.Entities, e2)
			msg.Text += s2

			s3 := "\n🔑 Owners\n"
			var e3 tgbotapi.MessageEntity
			e3.Type = "bold"
			e3.Offset = len(utf16.Encode([]rune(s1 + s2)))
			e3.Length = len(utf16.Encode([]rune(s3)))
			msg.Entities = append(msg.Entities, e3)
			msg.Text += s3

			var ss []models.Signer
			_ = json.Unmarshal([]byte(chat.Signers), &ss)
			for i, s := range ss {
				msg.Text += fmt.Sprintf("\n%d. @%s - %s\n", i+1, s.Name, s.Address)
			}

			var safeButton = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL("Link", e2.URL),
				),
			)
			msg.ReplyMarkup = safeButton
		case "verify":
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
			limit, _ := strconv.ParseInt(args, 10, 64)
			if limit == 0 {
				limit = 3
			}
			msg.Text = s.QueueTransaction(&msg, update.Message.Chat.ID, limit)
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

		case "setup":
			args := update.Message.CommandArguments()
			chainAndAddr := strings.Split(args, ":")
			if len(chainAndAddr) != 2 {
				msg.Text = "Wrong format"
			} else {
				s.SetupChat(update.Message.Chat.ID, update.Message.Chat.Title)
				ret := s.AddSafeWallet(update.Message.Chat.ID, chainAndAddr)
				if ret != "" {
					msg.Text = ret
				} else {
					msg.Text = fmt.Sprintf("Added safe wallet, address: %s", args)
				}
			}

		case "safestatus":
			msg.Text = s.Status(update.Message.Chat.ID)
		case "safequeue":
			msg.Text = s.GenerateQueueLink(update.Message.Chat.ID)
		case "safehistory":
			msg.Text = s.GenerateHistoryLink(update.Message.Chat.ID)
		case "ai":
			args := update.Message.CommandArguments()
			var request models.AIRequest
			request.Model = "text-davinci-003"
			request.Prompt = args
			request.Temperature = 0
			request.MaxTokens = 1000

			rs, _ := json.Marshal(request)
			resp, err := s.Client.R().
				SetHeader("Authorization", fmt.Sprintf("Bearer %s", "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")).
				SetHeader("Content-Type", "application/json").
				SetBody(string(rs)).
				Post("https://api.openai.com/v1/completions")
			if err != nil {
				msg.Text = err.Error()
			} else {
				var rsp models.AIResponse
				json.Unmarshal(resp.Body(), &rsp)
				msg.Text = rsp.Choices[0].Text
			}

		default:
			msg.Text = "I don't know that command"
		}

		if _, err := bot.Send(msg); err != nil {
			log.Panic(err)
		}
	}
}
