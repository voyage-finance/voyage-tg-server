package main

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-resty/resty/v2"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/voyage-finance/voyage-tg-server/config"
	"github.com/voyage-finance/voyage-tg-server/contracts/handlers"
	"github.com/voyage-finance/voyage-tg-server/http_server"
	"github.com/voyage-finance/voyage-tg-server/models"
	"github.com/voyage-finance/voyage-tg-server/service"
	"github.com/voyage-finance/voyage-tg-server/transaction/builder"
	"github.com/voyage-finance/voyage-tg-server/transaction/history"
	"github.com/voyage-finance/voyage-tg-server/transaction/queue"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"os"
	"strings"
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

	//db.AutoMigrate(&models.User{}, &models.Chat{}, &models.Signer{})
	//db.SetupJoinTable(&models.Chat{}, "Users", &models.Signer{})

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
		if update.Message == nil && update.CallbackQuery == nil { // ignore non-Message updates
			continue
		} else if update.Message == nil {
			// remember to send callback data with prefix - `/`
			update.Message = update.CallbackQuery.Message
			update.Message.Text = update.CallbackQuery.Data
			update.Message.From = update.CallbackQuery.From
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)

		if !strings.HasPrefix(update.Message.Text, "/") {
			continue
		}

		command := update.Message.Command()
		if update.CallbackQuery != nil {
			command = update.CallbackQuery.Data[1:]
		}

		// check safe availability
		go service.PingSafe(bot, update)

		// Extract the command from the Message.
		switch command {
		case "start":
			msg.ParseMode = "Markdown"
			startMessage := fmt.Sprintf("Welcome to the Voyage Safe bot!\n\n")
			startMessage += fmt.Sprintf("%s https://twitter.com/voyageOS\n\n", "*Twitter:*")
			startMessage += fmt.Sprintf("*How to setup?\n\n*")
			startMessage += fmt.Sprintf("1. Add the bot to your Telegram group with the button below;\n")
			startMessage += fmt.Sprintf("2. Use the /setup command to link your Telegram group to your Safe vault;\n")
			startMessage += fmt.Sprintf("3. Request your group members to /link their addresses to their Telegram accounts;\n")
			startMessage += fmt.Sprintf("4. Use the /help command for further assistance.\n\n")
			msg.Text = startMessage
			var button tgbotapi.InlineKeyboardMarkup
			if msg.ChatID > 0 {
				button = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonURL("Click here to add to a group", os.Getenv("SELF_INVITE")),
					),
				)
			} else {
				button = service.GetHelperButtons()
			}
			msg.ReplyMarkup = button
			var unsignedMessages []models.SignMessage
			db.Where("user_id = ?", update.Message.From.ID).Find(&unsignedMessages)
			for _, unsignedMessage := range unsignedMessages {
				fmt.Sprintf("Chat %v", unsignedMessage.ChatID)
				s.SendVerifyButton(bot, update, unsignedMessage)
			}
		case "info":
			chatId := update.Message.Chat.ID
			chat := s.QueryChat(chatId)
			msg.Text = "🔓 *Safe address*\n"
			msg.Text += fmt.Sprintf("\n`%s:%s`\n", chat.Chain, chat.SafeAddress)

			owners := s.Status(chatId)
			signerUsernames := s.GetOwnerUsernames(chat)

			msg.Text += fmt.Sprintf("\n🔑 *%d Owner(s)*\n", len(owners))

			for _, owner := range owners {
				username, ok := signerUsernames[strings.ToLower(owner)]
				if ok {
					msg.Text += fmt.Sprintf("*@%v* ", username)
				}
			}

			url := fmt.Sprintf("https://app.safe.global/%s:%s/home", chat.Chain, chat.SafeAddress)
			var safeButton = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL("Link", url),
				),
			)
			msg.ReplyMarkup = safeButton
			msg.ParseMode = "Markdown"
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
			var config tgbotapi.ChatAdministratorsConfig
			config.ChatID = update.Message.Chat.ID
			members, err := bot.GetChatAdministrators(config)
			if err != nil {
				msg.Text = err.Error()
				goto F
			}
			log.Printf("Members: %v", members)
			isAdmin := false
			for _, m := range members {
				if m.User.ID == update.Message.From.ID {
					isAdmin = true
				}
			}
			if !isAdmin {
				msg.Text = "You are not an admin of this chat."
				goto F
			}

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
		case "unlink":
			signMessage := s.GetOrCreateSignMessage(update.Message.Chat.ID, update.Message.From.ID, false)
			if !signMessage.IsVerified {
				msg.Text = fmt.Sprintf("You have not verified the message. Please send /link@%v", bot.Self.UserName)
				break
			}
			msg.Text = s.RemoveSigner(signMessage)
		case "request":
			requestHandler := builder.NewRequestHandler(update.Message.Chat.ID, s, update.Message.From.UserName)
			args := update.Message.CommandArguments()
			m, link := requestHandler.CreateRequest(args)
			if link != "" {
				startButton := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonURL("✍️ Submit it!", link),
					),
				)
				msg.ReplyMarkup = startButton
			}
			msg.Text = m
			msg.ParseMode = "Markdown"
		case "create":
			requestHandler := builder.NewCreateHandler(bot, update, update.Message.Chat.ID, s, update.Message.From.UserName)
			err := requestHandler.SendCreateDM()
			if err != "" {
				msg.Text = err
			} else {
				// Message to reply in chat. Adding conversation start button, in case if user does not have conversation with bot
				msg.Text = fmt.Sprintf("The transaction creation link was sent to Direct Message, *@%v*. If you do not see any message, then click the button below", update.Message.From.UserName)
				msg.ReplyToMessageID = update.Message.MessageID
				startButtonLink := fmt.Sprintf("https://t.me/%v", bot.Self.UserName)
				startButton := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonURL("Start conversation", startButtonLink),
					),
				)
				msg.ReplyMarkup = startButton
				msg.ParseMode = "Markdown"
			}
		case "stream":
			handler := handlers.NewLlamaHandler()

			// create stream
			payee := "0xEB8fb2f6D41706759B8544D5adA16FC710211ca2"
			llama := "0xA692FF8Fc672B513f7850C75465415437FE25617"
			amountPerSec := 38580246913580
			streamData, _ := handler.EncodeCreateStream(payee, int64(amountPerSec))
			creatStream := handler.EncodePacked(streamData, llama, 0)

			// approve
			erc20Handler := handlers.NewErc20Handler()
			spender := "0xA692FF8Fc672B513f7850C75465415437FE25617"
			// use ERC20 as example
			erc20 := "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174"
			value := 10000000000000
			approveData, _ := erc20Handler.EncodeApprove(spender, int64(value))
			approve := handler.EncodePacked(approveData, erc20, 0)

			// concat
			tns := append(approve, creatStream...)
			msg.Text = hexutil.Encode(tns)

		case "help":
			service.GetNavigationInstruction(bot, s, update.Message.Chat.ID, &msg)
			msg.ParseMode = "Markdown"
		default:
			msg.Text = "I don't know that command"
		}
	F:
		if _, err := bot.Send(msg); err != nil {
			log.Panic(err)
		}
	}
}
