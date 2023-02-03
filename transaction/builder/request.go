package builder

import (
	"fmt"
	"github.com/voyage-finance/voyage-tg-server/models"
	"github.com/voyage-finance/voyage-tg-server/service"
	"log"
	"os"
	"strconv"
	"strings"
)

type RequestHandler struct {
	chat      *models.Chat
	s         service.Service
	username  string
	amount    float64
	currency  string
	ownersMap map[string]string
	to        string
}

func NewRequestHandler(chatId int64, s service.Service, username string) *RequestHandler {
	chat := s.QueryChat(chatId)
	ownersMap := s.GetOwnerUsernames(chat)
	return &RequestHandler{chat: chat, s: s, username: username, ownersMap: ownersMap}
}

/*
	Validations
*/

func (requestHandler *RequestHandler) ValidateArgs(argString string) string {
	args := strings.Fields(argString)
	if len(args) != 2 {
		return "Must provide amount and currency! E.g: /request 1 $eth"
	}
	amount, currency := args[0], args[1]
	if currency[0:1] != "$" {
		return "Currency is not correct! E.g: $eth or $ETH"
	}
	amountFloat, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return err.Error()
	}
	requestHandler.amount = amountFloat
	requestHandler.currency = strings.ToLower(currency[1:])
	log.Println(requestHandler.amount)
	return ""
}

func (requestHandler *RequestHandler) ValidateSetup() string {

	toAddress := ""
	for addr, username := range requestHandler.ownersMap {
		if strings.ToLower(username) == strings.ToLower(requestHandler.username) {
			toAddress = addr
			break
		}
	}
	if toAddress == "" {
		return "You did not link your account. Please send /link"
	}
	requestHandler.to = toAddress
	return ""
}

func (requestHandler *RequestHandler) ValidateBalance() string {
	balances, err := requestHandler.s.GetBalance(requestHandler.chat)
	if err != nil {
		return err.Error()
	}
	log.Println(balances, requestHandler.currency)
	balance, ok := balances[requestHandler.currency]
	if !ok {
		return fmt.Sprintf("`$%s` balance is insufficient!", strings.ToUpper(requestHandler.currency))
	}
	if balance < requestHandler.amount {
		return fmt.Sprintf("Balance is insufficient! Max is %v $%v", balance, strings.ToUpper(requestHandler.currency))
	}
	return ""
}

func (requestHandler *RequestHandler) GetThreshold() service.StatusResp {
	return requestHandler.s.QuerySafeData(requestHandler.chat)
}

func (requestHandler *RequestHandler) CreateRequest(args string) (string, string) {
	errMsg := requestHandler.ValidateArgs(args)
	if errMsg != "" {
		return errMsg, ""
	}
	// 2.0 validate whether user setup account to address
	errMsg = requestHandler.ValidateSetup()
	if errMsg != "" {
		return errMsg, ""
	}
	// 3.0 validate user request
	errMsg = requestHandler.ValidateBalance()
	if errMsg != "" {
		return errMsg, ""
	}
	threshold := requestHandler.GetThreshold()
	response := fmt.Sprintf("🙏 *New Request!*\n\n")
	response += fmt.Sprintf("Transfer %v $%v\n\n", requestHandler.amount, strings.ToUpper(requestHandler.currency))
	response += fmt.Sprintf("To: `%v`\n\n", requestHandler.to)
	response += fmt.Sprintf("*Need %v submission(s) from:*\n", threshold.Threshold)
	for _, addr := range threshold.Owners {
		username, find := requestHandler.ownersMap[strings.ToLower(addr)]
		if find {
			response += fmt.Sprintf("*@%v* ", username)
		}
	}
	link := fmt.Sprintf("%v/safes/%v:%v/transactions/create/send?amount=%v&to=%v&currency=%v&chatId=%v",
		os.Getenv("FRONT_URL"),
		requestHandler.chat.Chain,
		requestHandler.chat.SafeAddress,
		requestHandler.amount,
		requestHandler.to,
		requestHandler.currency,
		requestHandler.chat.ChatId,
	)
	return response, link

}
