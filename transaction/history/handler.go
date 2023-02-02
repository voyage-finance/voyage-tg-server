package history

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/voyage-finance/voyage-tg-server/models"
	"github.com/voyage-finance/voyage-tg-server/service"
	common2 "github.com/voyage-finance/voyage-tg-server/transaction/common"
	"log"
)

type HistoryHandler struct {
	s service.Service
}

func NewQueuedHandler(s service.Service) *HistoryHandler {
	return &HistoryHandler{s: s}
}

func (h *HistoryHandler) Handle(id int64) string {
	chat := h.s.QueryChat(id)
	chainId := 1
	if chat.Chain == "matic" {
		chainId = 137
	}
	r := fmt.Sprintf("https://safe-client.safe.global/v1/chains/%v/safes/%v/transactions/history", chainId, common.HexToAddress(chat.SafeAddress))
	resp, err := h.s.Client.R().EnableTrace().Get(r)
	if err != nil {
		return err.Error()
	}
	var transactionResp common2.TransactionResponse
	json.Unmarshal(resp.Body(), &transactionResp)
	currentLastUpdatedNonce := h.s.GetCurrentLastConfirmedNonce(id)
	log.Printf("current last updated nonce: %d\n", currentLastUpdatedNonce)
	var updatedNonce int64
	for _, txn := range transactionResp.Results {
		if txn.Type == "TRANSACTION" && txn.TxInfo.Direction == "OUTGOING" {
			fmt.Println("Nonce: ", txn.ExecutionInfo.Nonce)
			if txn.ExecutionInfo.Nonce > currentLastUpdatedNonce {
				if updatedNonce == 0 {
					updatedNonce = txn.ExecutionInfo.Nonce
				}
				r = fmt.Sprintf("https://safe-client.safe.global/v1/chains/%d/transactions/%s", chainId, txn.Id)
				resp, err = h.s.Client.R().EnableTrace().Get(r)
				if err != nil {
					return err.Error()
				}
				var detailedTransaction common2.EachTransactionResponse
				json.Unmarshal(resp.Body(), &detailedTransaction)
				executor := detailedTransaction.DetailedExecutionInfo.Executor.Value
				log.Printf("executor: %s\n", executor)
				h.s.UpdatePoints(id, executor, 300)
				for _, s := range detailedTransaction.DetailedExecutionInfo.Signers {
					log.Printf("signer: %s\n", detailedTransaction.DetailedExecutionInfo.Signers)
					h.s.UpdatePoints(id, s.Value, 100)
				}
			}
		}
	}
	log.Printf("updated last confirm nonce: %d\n", updatedNonce)
	if updatedNonce != 0 {
		h.s.UpdateLastConfirmedNonce(id, updatedNonce)
	}
	ret := fmt.Sprintf("🏆 Leaderboard\n\n")

	// check after updating
	updateChat := h.s.QueryChat(id)
	var signers []models.Signer
	if updateChat.Signers != "" {
		err := json.Unmarshal([]byte(updateChat.Signers), &signers)
		if err != nil {
			log.Printf("RemoveSigner failed, error: %s\n", err.Error())
			return "Get current signer failed"
		}
	}
	for i, s := range signers {
		point := fmt.Sprintf("%d. %s - %d points\n", i+1, s.Name, s.Points)
		ret += point
	}
	return ret
}
