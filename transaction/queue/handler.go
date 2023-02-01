package queue

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/voyage-finance/voyage-tg-server/transaction"
	"log"
	"strings"
)

func (handler *QueuedHandler) HandleTransfer(transfer transaction.TxInfo, nonce int64) string {
	value := ""

	switch transfer.TransferInfo.Type {
	case TransferTypeNative:
		tokenValue := handler.s.ParseBalance(transfer.TransferInfo.Value, 18)
		tokenCurrency := handler.Currency
		value = fmt.Sprintf("Transfer %v `$%v`", tokenValue, tokenCurrency)
	case TransferTypeERC20:
		tokenValue := handler.s.ParseBalance(transfer.TransferInfo.Value, transfer.TransferInfo.Decimals)
		tokenCurrency := transfer.TransferInfo.TokenSymbol
		value = fmt.Sprintf("Transfer %v `$%v`", tokenValue, tokenCurrency)
	case TransferTypeERC721:
		value = fmt.Sprintf("Transfer %v `#%v`", transfer.TransferInfo.TokenSymbol, transfer.TransferInfo.TokenId)
	}

	value += fmt.Sprintf(" (nonce=`%v`):\n\n", nonce)
	value += fmt.Sprintf("*To*: `%v`\n\n", transfer.Recipient.Value)
	return value
}

func (handler *QueuedHandler) ResetConflictProgress() {
	handler.ConflictInProgress = false
	handler.ConflictCount = 0
	handler.ConflictTransaction = nil
}

func (handler *QueuedHandler) HandleCustom(custom transaction.Transaction) string {
	value := ""

	if custom.IsCancellation {
		value = fmt.Sprintf("Rejection (conflicts=`%v`)", handler.ConflictCount)
		handler.ResetConflictProgress()
		value += fmt.Sprintf(" (nonce=`%v`)\n\n", custom.ExecutionInfo.Nonce)

	} else {
		methodName := strings.ToUpper(custom.TxInfo.MethodName)
		amount := handler.s.ParseBalance(custom.TxInfo.Value, 18)
		currency := strings.ToUpper(custom.TxInfo.To.Name)
		appName := custom.SafeAppInfo.Name
		value = fmt.Sprintf("%v %v `%v` in app *%v*", methodName, amount, currency, appName)
		value += fmt.Sprintf(" (nonce=`%v`):\n\n", custom.ExecutionInfo.Nonce)
		value += fmt.Sprintf("*To*: `%v`\n\n", custom.TxInfo.To.Value)
	}
	return value
}

func (handler *QueuedHandler) HandleSettingsChange(settingChange transaction.TxInfo) string {
	value := ""
	switch settingChange.SettingsInfo.Type {
	case ChangeThreshold:
		value = fmt.Sprintf("Settings change: threshold update to %v\n\n", settingChange.SettingsInfo.Threshold)
	}
	return value
}

/*
Signers Confirmation section <<<<<
*/

func (handler *QueuedHandler) HandleConfirmations(id string, confirmationsRequired uint64, confirmationsSubmitted uint64) string {
	// signers/owners handling
	txRetrieveURL := fmt.Sprintf("https://safe-client.safe.global/v1/chains/%v/transactions/%v", handler.ChainId, id)
	resp, err := handler.s.Client.R().EnableTrace().Get(txRetrieveURL)
	confirmationResult := ""
	if err == nil {
		var eachTransactionResponse transaction.EachTransactionResponse
		json.Unmarshal(resp.Body(), &eachTransactionResponse)
		allSigners := map[string]string{}
		// run through signers of tx and store their usernames
		for _, signer := range eachTransactionResponse.DetailedExecutionInfo.Signers {
			signerValue := strings.ToLower(signer.Value)
			username, ok := handler.OwnerUsernames[signerValue]
			if ok && len(username) > 0 {
				allSigners[signerValue] += fmt.Sprintf("*@%s* ", username)
			}
		}
		confirmText := fmt.Sprintf("*Confirmations %v:*\n", len(eachTransactionResponse.DetailedExecutionInfo.Confirmations))
		confirmedSigners := map[string]bool{}
		for _, confirm := range eachTransactionResponse.DetailedExecutionInfo.Confirmations {
			signer := strings.ToLower(confirm.Signer.Value)
			lowerSigner, ok := allSigners[signer]
			if !ok {
				continue
			}
			confirmText += fmt.Sprintf("✅ %v \n", lowerSigner)
			confirmedSigners[signer] = true
		}
		confirmationResult += confirmText

		if confirmationsRequired-confirmationsSubmitted > 0 {
			unconfirmedText := fmt.Sprintf("\n*Need %v confirmation(s) from:*\n", confirmationsRequired-confirmationsSubmitted)
			index := 0
			for addr, username := range allSigners {
				_, ok := confirmedSigners[addr]
				if ok {
					continue
				}
				unconfirmedText += fmt.Sprintf("%s ", username)
				index++
			}
			if index == 0 {
				unconfirmedText = fmt.Sprintf("\n*Need %v confirmation(s) but no user is verified, so we cannot notify them!*", confirmationsRequired-confirmationsSubmitted)
			}

			confirmationResult += unconfirmedText + "\n\n"

		}
	}
	return confirmationResult
}

/*
	Signers Confirmation section end >>>>
*/

func (handler *QueuedHandler) GenerateSignLink(id string) string {
	link := fmt.Sprintf("https://app.safe.global/%s:%s/transactions/tx?id=%v", handler.Chain, common.HexToAddress(handler.SafeAddress), id)
	return fmt.Sprintf("[✍️ Sign/Submit it!](%v)\n\n----------\n", link)
}

func (handler *QueuedHandler) HandleTransaction(transaction transaction.Transaction) (string, bool) {
	value := ""
	switch transaction.TxInfo.Type {
	case TransactionTypeTransfer:
		value = handler.HandleTransfer(transaction.TxInfo, transaction.ExecutionInfo.Nonce)
	case TransactionTypeCustom:
		value = handler.HandleCustom(transaction)
	case TransactionTypeSettingsChange:
		value = handler.HandleSettingsChange(transaction.TxInfo)
	case TransactionTypeCreation:
		log.Println("NotImplemented! TransactionTypeCreation")
		return value, false
	}
	value += handler.HandleConfirmations(transaction.Id, transaction.ExecutionInfo.ConfirmationsRequired, transaction.ExecutionInfo.ConfirmationsSubmitted)
	value += handler.GenerateSignLink(transaction.Id)

	return value, true
}

func (handler *QueuedHandler) ResolveConflictType(transaction transaction.Transaction, conflictType string) *transaction.Transaction {
	// we need to store a transaction of Reject to show its content: owners, conflictCount, Nonce ....
	// if end of conflict, then return rejection tx data
	handler.ConflictCount++

	// need to store a reject tx (conflictType can be `End` or `HasNext`)
	if handler.ConflictTransaction == nil && transaction.TxInfo.IsCancellation {
		handler.ConflictTransaction = &transaction
	}
	// the last conflict tx with conflictType=`End`
	if conflictType == ConflictTypeEnd {
		return handler.ConflictTransaction
	}

	return nil
}

func (handler *QueuedHandler) Handle(id int64) string {
	chat := handler.s.QueryChat(id)
	chainId := 1
	chainCurrency := "ETH"
	if chat.Chain == "matic" {
		chainId = 137
		chainCurrency = "MATIC"
	}

	handler.Chain = chat.Chain
	handler.ChainId = chainId
	handler.Currency = chainCurrency
	handler.SafeAddress = chat.SafeAddress

	r := fmt.Sprintf("https://safe-client.safe.global/v1/chains/%v/safes/%v/transactions/queued?cursor=limit=10000&offset=0", chainId, common.HexToAddress(chat.SafeAddress))
	resp, err := handler.s.Client.R().EnableTrace().Get(r)
	if err != nil {
		return err.Error()
	}
	var queueTransactionResponse QueuedTransactionResponse
	json.Unmarshal(resp.Body(), &queueTransactionResponse)

	returnResponse := ""
	//startOffset := len(utf16.Encode([]rune(returnResponse)))
	counter := 1
	handler.OwnerUsernames = handler.s.GetOwnerUsernames(chat)

resultLoop:
	for i, result := range queueTransactionResponse.Results {
		switch result.Type {
		case ResultTypeTransaction:

			// 1.0 if current block is part of Rejection
			if handler.ConflictInProgress {
				conflictTx := handler.ResolveConflictType(result.Transaction, result.ConflictType)
				if conflictTx == nil {
					continue resultLoop
				}
				result.Transaction = *conflictTx
			}

			// 2.0 handle transaction
			txLine, isSupported := handler.HandleTransaction(result.Transaction)
			if !isSupported {
				continue resultLoop
			}
			returnResponse += fmt.Sprintf("%v) %v\n", counter, txLine)
			counter++

		case ResultTypeConflictHeader:
			handler.ConflictInProgress = true

		case ResultTypeLabel:
			continue

		case ResultTypeDataLabel:
			log.Println("Not implemented! ResultTypeLabel, index=", i)
		}
	}
	returnResponse = fmt.Sprintf("⏳ *Pending Transactions* (count=`%v`):\n\n\n", counter-1) + returnResponse

	return returnResponse
}
