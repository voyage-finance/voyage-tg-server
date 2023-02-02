package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/voyage-finance/voyage-tg-server/http_server"
	"github.com/voyage-finance/voyage-tg-server/transaction/builder"
	"github.com/voyage-finance/voyage-tg-server/transaction/history"
	"github.com/voyage-finance/voyage-tg-server/transaction/queue"
	"log"
	"os"
	"strings"
	"unicode/utf16"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/voyage-finance/voyage-tg-server/config"
	"github.com/voyage-finance/voyage-tg-server/models"
	"github.com/voyage-finance/voyage-tg-server/service"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	config.Init()
	dsn := fmt.Sprintf("host=%v "+
		"user=%v "+
		"password=%v "+
		"dbname=%v "+
		"port=%v "+
		"sslmode=disable "+
		"TimeZone=Asia/Almaty",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema
	//db.AutoMigrate(&models.User{})
	//db.AutoMigrate(&models.Chat{})
	//db.AutoMigrate(&models.SignMessage{})

	tokens, err := os.ReadFile("tokens.json")
	if err != nil {
		panic("failed to read tokens")
	}
	var ts []service.TokenInfo
	json.Unmarshal(tokens, &ts)

	client := resty.New()
	infuraAPI := "https://mainnet.infura.io/v3/" + os.Getenv("INFURA_KEY")
	ethClient, err := ethclient.Dial(infuraAPI)
	if err != nil {
		log.Fatal(err)
	}

	tokenInfo := make(map[string]service.TokenInfo)
	for _, t := range ts {
		tokenInfo[strings.ToLower(t.TokenAddress)] = t
	}

	s := service.Service{DB: db, Client: client, EthClient: ethClient, Tokens: tokenInfo}

	go http_server.HandleRequests(s)

	bot, err := tgbotapi.NewBotAPI(os.Getenv("BOT_API_KEY"))
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
		case "start":
			msg.ParseMode = "Markdown"
			startMessage := fmt.Sprintf("Welcome to the Voyage Safe Telegram bot!\n\n")
			startMessage += fmt.Sprintf("%s https://twitter.com/voyageOS\n\n", "*Twitter:*")
			startMessage += fmt.Sprintf("*Basic Commands\n*")
			startMessage += fmt.Sprintf("/this - Show safe vault info\n")
			startMessage += fmt.Sprintf("/setup - Sync a safe vault to this channel\n")
			startMessage += fmt.Sprintf("/link - Link your wallet address to your telegram account\n")
			startMessage += fmt.Sprintf("/unlink - unlink your wallet address from your telegram account\n")
			startMessage += fmt.Sprintf("/balance - check safe vault token balances\n")
			startMessage += fmt.Sprintf("/queue - show pending safe vault transactions\n")
			startMessage += fmt.Sprintf("/create - create a new safe vault transaction\n")
			startMessage += fmt.Sprintf("/request <amount> <token> - request funds from safe vault\n")
			startMessage += fmt.Sprintf("/leaderboard - show safe vault owners leaderboard\n\n")
			startMessage += fmt.Sprintf("You can add this bot to any group or use the commands above in this chat.\n\n")
			msg.Text = startMessage
			startButton := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL("Click here to add to a group", os.Getenv("SELF_INVITE")),
				),
			)
			msg.ReplyMarkup = startButton
			var unsignedMessages []models.SignMessage
			db.Where("user_id = ? AND is_verified = false", update.Message.From.ID).Find(&unsignedMessages)
			for _, unsignedMessage := range unsignedMessages {
				fmt.Sprintf("Chat %v", unsignedMessage.ChatID)
				s.SendVerifyButton(bot, update, unsignedMessage)
			}
		case "help":
			log.Printf("Chat id: %d\n", update.Message.Chat.ID)
			msg.Text = `Commands:
					/this: show safe vault info
					/setup: sync a safe vault to this channel
					/link:  link your wallet address to your telegram account
					/unlink: unlink your wallet address from your telegram account
					/balance: check safe vault token balances
					/queue: show pending safe vault transactions
					/request: request funds from safe vault (e.g. /request 1 $matic)
					/leaderboard: show safe vault owners leaderboard
			`
		case "this":
			chatId := update.Message.Chat.ID
			chat := s.QueryChat(chatId)
			s1 := "🔓 Safe address\n"
			msg.Text = s1
			log.Println(chat)
			// 1. Safe address should be bold
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

			startOffset := len(utf16.Encode([]rune(msg.Text)))

			var ss []models.Signer
			_ = json.Unmarshal([]byte(chat.Signers), &ss)
			for i, s := range ss {
				n := fmt.Sprintf("\n%d. @%s - ", i+1, s.Name)
				msg.Text += n
				a := fmt.Sprintf("%s\n", s.Address)
				var e tgbotapi.MessageEntity
				e.Type = "code"
				e.Offset = startOffset + len(utf16.Encode([]rune(n)))
				e.Length = len(utf16.Encode([]rune(a)))
				msg.Entities = append(msg.Entities, e)
				msg.Text += a
				startOffset += len(utf16.Encode([]rune(n)))
				startOffset += len(utf16.Encode([]rune(a)))
			}

			var safeButton = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL("Link", e2.URL),
				),
			)
			msg.ReplyMarkup = safeButton
		case "link":
			s.SetupChat(update.Message.Chat.ID, update.Message.Chat.Title, update.Message.From.ID, update.Message.From.UserName)
			signMessage := s.GetOrCreateSignMessage(update.Message.Chat.ID, update.Message.From.ID, false)
			// Message of Direct Message
			s.SendVerifyButton(bot, update, signMessage)

			// Message to reply in chat. Adding conversation start button, in case if user does not have conversation with bot
			msg.Text = fmt.Sprintf("Please verify, @%v, your account address via Sign-In With Ethereum. "+
				"The message was sent to Direct Message. If you do not see any message, then click the button below", update.Message.From.UserName)
			msg.ReplyToMessageID = update.Message.MessageID
			startButtonLink := fmt.Sprintf("https://t.me/%v", bot.Self.UserName)
			startButton := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL("Start conversation", startButtonLink),
				),
			)
			msg.ReplyMarkup = startButton
		case "queue":
			queueHandler := queue.NewQueuedHandler(s)
			msg.Text = queueHandler.Handle(update.Message.Chat.ID)
			msg.ParseMode = "Markdown"
		case "leaderboard":
			historyHandler := history.NewQueuedHandler(s)
			msg.Text = historyHandler.Handle(update.Message.Chat.ID)
			msg.ParseMode = "Markdown"
		case "balance":
			chatId := update.Message.Chat.ID
			msg.Text = s.QueryTokenBalance(chatId)
			var e tgbotapi.MessageEntity
			e.Type = "bold"
			e.Offset = 2
			e.Length = 16
			msg.Entities = append(msg.Entities, e)
		case "setup":
			s.SetupChat(update.Message.Chat.ID, update.Message.Chat.Title, update.Message.From.ID, update.Message.From.UserName)
			signMessage := s.GetOrCreateSignMessage(update.Message.Chat.ID, update.Message.From.ID, true)
			// Message of Direct Message
			s.SendLinkButton(bot, update, signMessage)

			// Message to reply in chat. Adding conversation start button, in case if user does not have conversation with bot
			msg.Text = fmt.Sprintf("Please sign message via Sign-In With Ethereum, *@%v*, and choose your Safe Account to link. "+
				"The message was sent to Direct Message. If you do not see any message, then click the button below", update.Message.From.UserName)
			msg.ReplyToMessageID = update.Message.MessageID
			startButtonLink := fmt.Sprintf("https://t.me/%v", bot.Self.UserName)
			startButton := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL("Start conversation", startButtonLink),
				),
			)
			msg.ReplyMarkup = startButton
			msg.ParseMode = "Markdown"
		case "ai":
			args := update.Message.CommandArguments()
			var request models.AIRequest
			request.Model = "text-davinci-003"
			request.Prompt = args
			request.Temperature = 0
			request.MaxTokens = 1000

			rs, _ := json.Marshal(request)
			resp, err := s.Client.R().
				SetHeader("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("OPENAI_TOKEN"))).
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
		case "unlink":
			signMessage := s.GetOrCreateSignMessage(update.Message.Chat.ID, update.Message.From.ID, false)
			if !signMessage.IsVerified {
				msg.Text = fmt.Sprintf("You have not verified the message. Please send /setup@%v", bot.Self.UserName)
				break
			}
			msg.Text = s.RemoveSigner(signMessage, update.Message.From.UserName)
		case "request":
			requestHandler := builder.NewRequestHandler(update.Message.Chat.ID, s, update.Message.From.UserName)
			args := update.Message.CommandArguments()
			m, link := requestHandler.CreateRequest(args)
			startButton := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL("✍️ Submit it!", link),
				),
			)
			msg.ReplyMarkup = startButton
			msg.Text = m
			msg.ParseMode = "Markdown"
		default:
			msg.Text = "I don't know that command"
		}

		if _, err := bot.Send(msg); err != nil {
			log.Panic(err)
		}
	}
}
