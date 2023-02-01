package queue

import (
	"github.com/voyage-finance/voyage-tg-server/models"
	"github.com/voyage-finance/voyage-tg-server/service"
	"github.com/voyage-finance/voyage-tg-server/transaction/common"
)

type AddressValue struct {
	Value string
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
	Chat                *models.Chat
	ConflictTransaction *common.Transaction
}

func NewQueuedHandler(s service.Service) *QueuedHandler {
	return &QueuedHandler{s: s}
}
