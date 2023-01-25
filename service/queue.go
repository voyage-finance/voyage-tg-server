package service

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/voyage-finance/voyage-tg-server/models"
	"log"
	"strings"
)

type QueueTransactionResponse struct {
	Next     int64        `json:"next"`
	Previous int64        `json:"previous"`
	Results  []ResultType `json:"results"`
}

type ResultType struct {
	Type         string            `json:"type"`
	Transaction  QueuedTransaction `json:"transaction"`
	ConflictType string            `json:"conflictType"`
	Nonce        int64             `json:"nonce"`
}

type QueuedTransaction struct {
	Id            string
	Timestamp     uint64
	TxStatus      string        `json:"txStatus"`
	TxInfo        TxInfo        `json:"txInfo"`
	ExecutionInfo ExecutionInfo `json:"executionInfo"`
}

type TxInfo struct {
	Type         string
	Sender       AddressValue
	Recipient    AddressValue
	Direction    string
	TransferInfo TransferInfo `json:"transferInfo"`
}

type AddressValue struct {
	Value string
}

type TransferInfo struct {
	Type        string
	Value       string
	TokenSymbol string `json:"tokenSymbol"`
}

type ExecutionInfo struct {
	Type                   string
	Nonce                  int64
	ConfirmationsRequired  uint64 `json:"confirmationsRequired"`
	ConfirmationsSubmitted uint64 `json:"confirmationsSubmitted"`
}

// MULTISIG Retrieve

type EachTransactionResponse struct {
	DetailedExecutionInfo DetailedExecutionInfo
}

type DetailedExecutionInfo struct {
	Type          string
	SubmittedAt   uint64 `json:"submittedAt"`
	Signers       []AddressValue
	Confirmations []ConfirmationSigner
}
type ConfirmationSigner struct {
	Signer      AddressValue
	Signature   string
	SubmittedAt uint64 `json:"submittedAt"`
}

// constants
var NativeToken = map[interface{}]string{
	1:   "Eth",
	137: "Matic",
}

var TransactionType = "TRANSACTION"
var ConflictType = "CONFLICT_HEADER"

func (s *Service) GetOwnerUsernames(chat *models.Chat) map[string]string {
	var result = map[string]string{}
	var signers []models.Signer
	if chat.Signers != "" {
		err := json.Unmarshal([]byte(chat.Signers), &signers)
		if err != nil {
			log.Printf("Cannot get Signer in Queue request: %s\n", err.Error())
			return map[string]string{}
		}
	}
	for _, signer := range signers {
		result[strings.ToLower(signer.Address)] = signer.Name
	}
	return result
}

func GetRejectMessage(counter int, result ResultType, conflictCount int, chat *models.Chat) string {
	line1 := fmt.Sprintf("%v) Rejection (nonce=`%v`), (conflicts=`%v`)\n\n", counter, result.Transaction.ExecutionInfo.Nonce, conflictCount+1)
	link := fmt.Sprintf("https://app.safe.global/%s:%s/transactions/tx?id=%v", chat.Chain, common.HexToAddress(chat.SafeAddress), result.Transaction.Id)
	line2 := fmt.Sprintf("[✍️ Sign/Submit it!](%v)\n-----------------------------------------------\n", link)
	return line1 + line2
}

func (s *Service) QueueTransactionV2(m *tgbotapi.MessageConfig, id int64) string {
	chat := s.QueryChat(id)
	network := 1
	if chat.Chain == "matic" {
		network = 137
	}
	r := fmt.Sprintf("https://safe-client.safe.global/v1/chains/%v/safes/%v/transactions/queued", network, common.HexToAddress(chat.SafeAddress))
	resp, err := s.Client.R().EnableTrace().Get(r)
	if err != nil {
		return err.Error()
	}

	var queueTransactionResponse QueueTransactionResponse
	json.Unmarshal(resp.Body(), &queueTransactionResponse)

	returnResponse := ""
	//startOffset := len(utf16.Encode([]rune(returnResponse)))
	counter := 1
	ownerUsernames := s.GetOwnerUsernames(chat)
	isConflicted := false
	conflictCount := 0
	hasRejectTx := false
	for _, result := range queueTransactionResponse.Results {
		if result.Type == ConflictType {
			// if header says that it is conflict, then initialize conflict logic
			isConflicted = true
			continue
		}
		if result.Type != TransactionType {
			continue
		}
		txType := result.Transaction.TxInfo.Type

		if result.ConflictType == "HasNext" && txType != "Custom" {
			// conflicted tx
			conflictCount++
			continue
		}

		// transaction info
		eachTxResponse := ""
		switch txType {
		case "Transfer":
			fromAddress := result.Transaction.TxInfo.Sender.Value
			toAddress := result.Transaction.TxInfo.Recipient.Value
			currency := NativeToken[network]
			if result.Transaction.TxInfo.TransferInfo.Type != "NATIVE_COIN" {
				currency = result.Transaction.TxInfo.TransferInfo.TokenSymbol
			}

			value := s.ParseBalance(result.Transaction.TxInfo.TransferInfo.Value, 18)

			// execution info
			nonce := result.Transaction.ExecutionInfo.Nonce
			confirmationsRequired := result.Transaction.ExecutionInfo.ConfirmationsRequired
			confirmationsSubmitted := result.Transaction.ExecutionInfo.ConfirmationsSubmitted

			// conflict end handling
			conflictMsg := ":\n"
			if isConflicted && result.ConflictType == "End" {
				if hasRejectTx {
					returnResponse += GetRejectMessage(counter, result, conflictCount, chat)
					isConflicted = false
					conflictCount = 0
					hasRejectTx = false
					counter += 1
					break
				}
				conflictMsg = fmt.Sprintf(", (conflicts=`%v`):\n", conflictCount)
				isConflicted = false
				conflictCount = 0
			}

			line1 := fmt.Sprintf("%v) Transfer %v `$%v` (nonce=`%v`)%v", counter, value, strings.ToUpper(currency), nonce, conflictMsg)
			line2 := fmt.Sprintf("\nFrom: `%v`\nTo: `%v`\n", fromAddress, toAddress)
			line3 := fmt.Sprintf("\nSigning Threshold: %v/%v\n", confirmationsSubmitted, confirmationsRequired)
			eachTxResponse += line1 + line2 + line3

			// signers/owners handling
			txRetrieveURL := fmt.Sprintf("https://safe-client.safe.global/v1/chains/%v/transactions/%v", network, result.Transaction.Id)
			resp, err := s.Client.R().EnableTrace().Get(txRetrieveURL)
			if err == nil {
				var eachTransactionResponse EachTransactionResponse
				json.Unmarshal(resp.Body(), &eachTransactionResponse)
				allSigners := map[string]string{}
				// run through signer of tx and store thir usernames
				for _, signer := range eachTransactionResponse.DetailedExecutionInfo.Signers {
					signerValue := strings.ToLower(signer.Value)
					username, ok := ownerUsernames[signerValue]
					allSigners[signerValue] = fmt.Sprintf("`%v` ", signerValue)
					if ok && len(username) > 0 {
						allSigners[signerValue] += fmt.Sprintf("- *@%s* ", username)
					}
				}
				confirmText := "\nConfirmations:\n"
				confirmedSigners := map[string]bool{}
				for index, confirm := range eachTransactionResponse.DetailedExecutionInfo.Confirmations {
					signer := strings.ToLower(confirm.Signer.Value)
					confirmText += fmt.Sprintf("%v. %v \n", index+1, allSigners[signer])
					confirmedSigners[signer] = true
				}
				eachTxResponse += confirmText

				if confirmationsRequired-confirmationsSubmitted > 0 {
					unconfirmedText := fmt.Sprintf("\nNeed %v confirmation(s) from:\n", confirmationsRequired-confirmationsSubmitted)
					index := 1
					for addr, username := range allSigners {
						_, ok := confirmedSigners[addr]
						if ok {
							continue
						}
						unconfirmedText += fmt.Sprintf("%v. %v\n", index, username)
						index++
					}
					eachTxResponse += unconfirmedText + "\n"

				}

			} else {
				log.Println("Error in retrieving each tx info: ", err.Error())
			}

			// link to tx
			link := fmt.Sprintf("https://app.safe.global/%s:%s/transactions/tx?id=%v", chat.Chain, common.HexToAddress(chat.SafeAddress), result.Transaction.Id)
			line4 := fmt.Sprintf("[✍️ Sign/Submit it!](%v)\n-----------------------------------------------\n", link)
			eachTxResponse += line4

			returnResponse += eachTxResponse
			counter += 1
		case "Custom":
			// conflict end handling
			fmt.Println("in Custom side", isConflicted, result)
			eachTxResponse := ""
			if isConflicted && result.ConflictType == "End" {
				eachTxResponse = GetRejectMessage(counter, result, conflictCount, chat)
				isConflicted = false
				conflictCount = 0
				hasRejectTx = false
				counter += 1
			} else if isConflicted {
				hasRejectTx = true
			}
			returnResponse += eachTxResponse
		default:
			log.Println("Default was called")
		}
	}
	returnResponse = fmt.Sprintf("*Pending Transactions* (count=`%v`):\n\n", counter-1) + returnResponse
	return returnResponse
}
