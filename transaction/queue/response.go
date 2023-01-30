package queue

import (
	"github.com/voyage-finance/voyage-tg-server/service"
	"github.com/voyage-finance/voyage-tg-server/transaction"
)

type AddressValue struct {
	Value string
}

type QueuedTransactionResponse struct {
	Next     int64 `json:"next"`
	Previous int64 `json:"previous"`
	Results  []struct {
		Type                    string `json:"type"`
		ConflictType            string `json:"conflictType"`
		Nonce                   int64  `json:"nonce"`
		transaction.Transaction `json:"transaction"`
	} `json:"results"`
}

func (r *QueuedTransactionResponse) TransferHandle() {

}

type QueuedHandler struct {
	s           service.Service
	ChainId     int
	Chain       string
	SafeAddress string

	Currency           string
	ConflictInProgress bool
	ConflictCount      int

	OwnerUsernames      map[string]string
	ConflictTransaction *transaction.Transaction
}

func NewQueuedHandler(s service.Service) *QueuedHandler {
	return &QueuedHandler{s: s}
}
