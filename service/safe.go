package service

import (
	"encoding/json"
	"fmt"
	"github.com/voyage-finance/voyage-tg-server/models"
	"log"
	"strings"
	"unicode/utf16"

	"github.com/dustin/go-humanize"
	"github.com/ethereum/go-ethereum/common"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/shopspring/decimal"
)

type QueueTransactionResp struct {
	Count int64              `json:"count"`
	Next  string             `json:"next"`
	QT    []QueueTransaction `json:"results"`
}

type QueueTransaction struct {
	Safe                  string
	To                    string
	Value                 string
	Data                  *string
	Operation             int64
	GasToken              string
	SafeTxGas             int64
	BaseGas               int64
	GasPrice              string
	RefundReceiver        string
	Nonce                 int64
	ExecutionDate         *string
	SubmissionDate        *string
	Modified              *string
	SafeTxHash            *string
	IsExecuted            bool
	DataDecoded           DataDecoded
	ConfirmationsRequired int64
	Confirmations         []TransactionConfirmation
	TxType                string
}

type TransactionConfirmation struct {
	Owner           string
	SubmissionDate  string
	TransactionHash *string
	Signature       string
	SignatureType   string
}

type DataDecoded struct {
	Method     string
	Parameters []Parameter
}

type Parameter struct {
	Name  string
	Type  string
	Value string
}

type TokenBalance struct {
	TokenAddress string
	Token        Token
	Balance      string
}

type Token struct {
	Name     string
	Symbol   string
	Decimals int64
	LogoUri  string
}

type Currency struct {
	Id     string `json:"id"`
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
}

func (p *Parameter) String() string {
	return fmt.Sprintf("[Name: %s, Type: %s, Value: %s]", p.Name, p.Type, p.Value)
}

func (s *Service) GetBalance(chat *models.Chat) (map[string]float64, error) {
	network := "mainnet"
	if chat.Chain == "matic" {
		network = "polygon"
	}
	r := fmt.Sprintf("https://safe-transaction-%s.safe.global/api/v1/safes/%s/balances/?trusted=false&exclude_spam=false", network, chat.SafeAddress)
	resp, err := s.Client.R().EnableTrace().Get(r)
	if err != nil {
		return nil, err
	}
	var balances []TokenBalance
	json.Unmarshal(resp.Body(), &balances)
	supportedCurrencies, err := s.Client.R().EnableTrace().Get("https://api.coingecko.com/api/v3/coins/list")
	if err != nil {
		return nil, err
	}
	var currencies []Currency
	json.Unmarshal(supportedCurrencies.Body(), &currencies)
	m := make(map[string]bool)
	for _, c := range currencies {
		m[c.Symbol] = true
	}

	balanceMap := map[string]float64{}
	for _, balance := range balances {
		if balance.Token.Symbol == "" {
			if network == "mainnet" {
				balance.Token.Symbol = "ETH"
			}
			if network == "polygon" {
				balance.Token.Symbol = "MATIC"
			}

		}
		if balance.Token.Decimals == 0 {
			balance.Token.Decimals = 18
		}
		symbol := strings.ToLower(balance.Token.Symbol)
		if m[symbol] {
			formatBalance, _ := decimal.NewFromString(balance.Balance)
			formatBalance = formatBalance.Shift(0 - int32(balance.Token.Decimals))
			formatBalance = formatBalance.Truncate(4)
			fValue, _ := formatBalance.Float64()
			balanceMap[symbol] = fValue
		}
	}
	return balanceMap, nil
}

func (s *Service) QueryTokenBalance(id int64) string {
	chat := s.QueryChat(id)

	balanceMap, err := s.GetBalance(chat)
	if err != nil {
		return err.Error()
	}
	ret := "💰 Account Balance\n"
	index := 1
	for symbol, balance := range balanceMap {

		hValue := humanize.Commaf(balance)
		ret += fmt.Sprintf(`
			%d. $%s - %s
			`, index, strings.ToUpper(symbol), hValue)
		index++

	}
	return ret

}

func (s *Service) ParseBalance(bal string, decimals int32) string {
	formatBalance, _ := decimal.NewFromString(bal)
	formatBalance = formatBalance.Shift(0 - decimals)
	formatBalance = formatBalance.Truncate(4)
	fValue, _ := formatBalance.Float64()
	hValue := humanize.Commaf(fValue)
	return hValue

}

func (s *Service) SerializeBalance(bal string, decimals int32) string {
	formatBalance, _ := decimal.NewFromString(bal)
	formatBalance = formatBalance.Shift(decimals)
	return formatBalance.String()

}

func (s *Service) QueueTransaction(m *tgbotapi.MessageConfig, id int64, limit int64) string {
	chat := s.QueryChat(id)
	network := "mainnet"
	if chat.Chain == "matic" {
		network = "polygon"
	}
	r := fmt.Sprintf("https://safe-transaction-%s.safe.global/api/v1/safes/%s/all-transactions/?limit=%d&executed=false&queued=true&trusted=true", network, common.HexToAddress(chat.SafeAddress), limit)
	log.Printf("request: %s\n", r)
	resp, err := s.Client.R().EnableTrace().Get(r)
	if err != nil {
		return err.Error()
	}

	var queueTransactionResp QueueTransactionResp
	json.Unmarshal(resp.Body(), &queueTransactionResp)
	ret := ""
	s1 := "Pending Transactions: \n"
	ret += s1
	startOffset := len(utf16.Encode([]rune(s1)))
	index := 1
	for _, qt := range queueTransactionResp.QT {
		parameters := ""
		for _, p := range qt.DataDecoded.Parameters {
			parameters += p.String()
		}
		safeTxHash := ""
		if qt.SafeTxHash != nil {
			safeTxHash = *qt.SafeTxHash
		}
		link := fmt.Sprintf("https://app.safe.global/%s:%s/transactions/tx?id=multisig_%s_%s", chat.Chain, common.HexToAddress(chat.SafeAddress), common.HexToAddress(chat.SafeAddress), safeTxHash)

		if !qt.IsExecuted && qt.TxType == "MULTISIG_TRANSACTION" {
			tokenInfo, exist := s.Tokens[strings.ToLower(qt.To)]
			// erc20 transfer
			if exist {
				if tokenInfo.TokenType == "ERC20" {
					var to string
					var amount string
					for _, p := range qt.DataDecoded.Parameters {
						if p.Name == "to" {
							to = p.Value
						} else if p.Name == "value" {
							amount = p.Value
						}
					}

					s2 := fmt.Sprintf("\n%d Transfer %s $%s to ", index, s.ParseBalance(amount, int32(tokenInfo.Decimals)), strings.ToUpper(tokenInfo.Symbol))
					ret += s2
					a2 := fmt.Sprintf("%s\n", to)
					ret += a2
					var as tgbotapi.MessageEntity
					as.Type = "code"
					as.Offset = len(utf16.Encode([]rune(s2))) + startOffset
					as.Length = len(utf16.Encode([]rune(a2)))
					m.Entities = append(m.Entities, as)

					startOffset += len(utf16.Encode([]rune(a2 + s2)))

					s3 := fmt.Sprintf("\nSigning Threshold: %d/%d\n", len(qt.Confirmations), qt.ConfirmationsRequired)
					ret += s3
					s4 := fmt.Sprintln("✍️ Sign/Submit it!")
					ret += s4
					var e tgbotapi.MessageEntity
					e.Type = "text_link"
					e.URL = link
					e.Offset = len(utf16.Encode([]rune(s3))) + startOffset
					e.Length = len(utf16.Encode([]rune(s4)))
					m.Entities = append(m.Entities, e)

					startOffset += len(utf16.Encode([]rune(s3 + s4)))

				}
			} else {
				// todo could be native transfer or alt coin transfer
				// indicate it is a native transaction
				s2 := fmt.Sprintf("\n%d Transfer %s $%s to ", index, s.ParseBalance(qt.Value, 18), strings.ToUpper(chat.Chain))
				ret += s2
				a2 := fmt.Sprintf("%s\n", qt.To)
				ret += a2
				var as tgbotapi.MessageEntity
				as.Type = "code"
				as.Offset = len(utf16.Encode([]rune(s2))) + startOffset
				as.Length = len(utf16.Encode([]rune(a2)))
				m.Entities = append(m.Entities, as)

				startOffset += len(utf16.Encode([]rune(a2 + s2)))

				s3 := fmt.Sprintf("\nSigning Threshold: %d/%d\n", len(qt.Confirmations), qt.ConfirmationsRequired)
				ret += s3
				s4 := fmt.Sprintln("✍️ Sign/Submit it!")
				ret += s4
				var e tgbotapi.MessageEntity
				e.Type = "text_link"
				e.URL = link
				e.Offset = len(utf16.Encode([]rune(s3))) + startOffset
				e.Length = len(utf16.Encode([]rune(s4)))
				m.Entities = append(m.Entities, e)

				startOffset += len(utf16.Encode([]rune(s3 + s4)))
			}
			index++
		}
	}

	return ret

}

func (s *Service) GenerateQueueLink(id int64) string {
	chat := s.QueryChat(id)
	return fmt.Sprintf("https://app.safe.global/%s/transactions/queue", chat.SafeAddress)
}

func (s *Service) GenerateHistoryLink(id int64) string {
	chat := s.QueryChat(id)
	return fmt.Sprintf("Track transaction history here: https://app.safe.global/%s/transactions/history", chat.SafeAddress)
}

type StatusResp struct {
	Address   string   `json:"address"`
	Nonce     int64    `json:"nonce"`
	Threshold int64    `json:"threshold"`
	Owners    []string `jsom:"owners"`
}

func (s *Service) QuerySafeData(chat *models.Chat) StatusResp {
	network := "mainnet"
	if chat.Chain == "matic" {
		network = "polygon"
	}
	r := fmt.Sprintf("https://safe-transaction-%s.safe.global/api/v1/safes/%s/", network, chat.SafeAddress)
	resp, _ := s.Client.R().EnableTrace().Get(r)

	var statusResp StatusResp
	_ = json.Unmarshal(resp.Body(), &statusResp)
	return statusResp
}

func (s *Service) Status(chatId int64) []string {
	chat := s.QueryChat(chatId)
	statusResp := s.QuerySafeData(chat)
	// make lower cased
	for k, v := range statusResp.Owners {
		// to modify the value at index k in the slice
		// we assign the new value to names[k]
		statusResp.Owners[k] = strings.ToLower(v)
	}
	return statusResp.Owners
}

func (s *Service) GetOwnerUsernames(chat *models.Chat) map[string]string {
	var result = map[string]string{}
	var signers []models.Signer
	s.DB.Preload("User").Find(&signers, "chat_chat_id = ?", chat.ChatId)
	for _, signer := range signers {
		result[strings.ToLower(signer.Address)] = strings.ToLower(signer.User.UserName)
	}
	log.Println(result, "result")
	return result
}

func (s *Service) GetChainId(chain string) int {
	chain = strings.ToLower(chain)
	switch chain {
	case "matic":
		return 137
	case "polygon":
		return 137
	}
	return 1
}

func (s *Service) GetSafeChain(chainId int) string {
	switch chainId {
	case 137:
		return "matic"
	}
	return "eth"
}

func (s *Service) GetAddressByUsername(chat *models.Chat, username string) string {
	signers := s.GetOwnerUsernames(chat)
	username = strings.ToLower(username)
	for addr, user := range signers {
		if user == username {
			return addr
		}
	}
	return ""
}
