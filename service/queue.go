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
	Type  string
	Value string
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

	returnResponse := "*Pending Transactions:*\n"
	//startOffset := len(utf16.Encode([]rune(returnResponse)))
	counter := 1
	ownerUsernames := s.GetOwnerUsernames(chat)
	log.Println("ownerUsernames: ", ownerUsernames)
	isConflicted := false
	for _, result := range queueTransactionResponse.Results {
		if result.Type == ConflictType {
			// conflict start handling
			returnResponse += "\n`Transactions with Conflicts started <<<<<<<<<<<<<<< `\n\n"
			if result.Nonce != -1 {
				returnResponse += fmt.Sprintf("Reason: %v These transactions conflict as they use the same nonce. Executing one will automatically replace the other(s). [Learn more](https://help.safe.global/en/articles/4730252-why-are-transactions-with-the-same-nonce-conflicting-with-each-other)\n\n", result.Nonce)
			}
			isConflicted = true
			continue
		}
		if result.Type != TransactionType {
			continue
		}

		// transaction info
		txType := result.Transaction.TxInfo.Type
		eachTxResponse := ""
		switch txType {
		case "Transfer":
			fromAddress := result.Transaction.TxInfo.Sender.Value
			toAddress := result.Transaction.TxInfo.Recipient.Value
			currency := NativeToken[network]
			if result.Transaction.TxInfo.TransferInfo.Type != "NATIVE_COIN" {
				currency = result.Transaction.TxInfo.TransferInfo.Type
			}
			if result.ConflictType == "HasNext" {
				// conflicted tx
			}

			value := s.ParseBalance(result.Transaction.TxInfo.TransferInfo.Value, 18)

			// execution info
			nonce := result.Transaction.ExecutionInfo.Nonce
			confirmationsRequired := result.Transaction.ExecutionInfo.ConfirmationsRequired
			confirmationsSubmitted := result.Transaction.ExecutionInfo.ConfirmationsSubmitted

			line1 := fmt.Sprintf("%v) Transfer %v `$%v` (nonce=`%v`):\n", counter, value, strings.ToUpper(currency), nonce)
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
						allSigners[signerValue] += fmt.Sprintf("- *@%s*  👆", username)
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
					unconfirmedText := "\nNeed confirmations from:\n"
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
			line4 := fmt.Sprintf("[✍️ Sign/Submit it!](%v)\n\n", link)
			eachTxResponse += line4

			// conflict end handling
			if isConflicted && result.ConflictType == "End" {
				eachTxResponse += "`Transactions with Conflicts ended >>>>>>>>>>>>>>> `\n\n"
				isConflicted = false

			}

			returnResponse += eachTxResponse
			counter += 1
		}
	}
	return returnResponse
}
