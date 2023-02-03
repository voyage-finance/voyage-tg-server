package builder

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/voyage-finance/voyage-tg-server/models"
	"github.com/voyage-finance/voyage-tg-server/service"
	"golang.org/x/exp/slices"
	"log"
	"os"
	"strings"
)

type CreateHandler struct {
	chat     *models.Chat
	s        service.Service
	update   tgbotapi.Update
	bot      *tgbotapi.BotAPI
	username string
	address  string
}

func NewCreateHandler(bot *tgbotapi.BotAPI, update tgbotapi.Update, chatId int64, s service.Service, username string) *CreateHandler {
	chat := s.QueryChat(chatId)

	return &CreateHandler{bot: bot, update: update, chat: chat, s: s, username: strings.ToLower(username)}
}

/*
	Validations
*/

func (createHandler *CreateHandler) ValidateSetup() string {
	ownersMap := createHandler.s.GetOwnerUsernames(createHandler.chat)

	address := ""
	for addr, name := range ownersMap {
		if name == createHandler.username {
			address = addr
		}
	}
	if address == "" {
		return "You did not setup your account. Please send /link"
	}
	createHandler.address = address
	return ""
}

func (createHandler *CreateHandler) ValidateAdmin() string {
	owners := createHandler.s.Status(createHandler.chat.ChatId)

	if !slices.Contains(owners, createHandler.address) {
		return "You are not in list of owners. Send /this to see all owners list"
	}

	return ""
}

func (createHandler *CreateHandler) SendCreateDM() string {
	// 1.0 validate whether user setup account to address
	errMsg := createHandler.ValidateSetup()
	if errMsg != "" {
		return errMsg
	}
	// 2.0 validate user admin or not
	errMsg = createHandler.ValidateAdmin()
	if errMsg != "" {
		return errMsg
	}
	dmText := fmt.Sprintf("Press button below to create transaction with Safe=`%v`. This link was sent from chat=*%v*",
		createHandler.chat.SafeAddress, createHandler.chat.Title)

	dmMsg := tgbotapi.NewMessage(createHandler.update.Message.From.ID, dmText)

	link := fmt.Sprintf("%v/safes/%v:%v/transactions/create?&chatId=%v",
		os.Getenv("FRONT_URL"),
		createHandler.chat.Chain,
		createHandler.chat.SafeAddress,
		createHandler.chat.ChatId,
	)
	var safeButton = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("Create transaction", link),
		),
	)

	dmMsg.ReplyMarkup = safeButton
	dmMsg.ParseMode = "Markdown"
	if _, err := createHandler.bot.Send(dmMsg); err != nil {
		log.Println(err)
	}
	return ""

}
