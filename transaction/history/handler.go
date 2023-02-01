package history

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/voyage-finance/voyage-tg-server/service"
	common2 "github.com/voyage-finance/voyage-tg-server/transaction/common"
)

type HistoryHandler struct {
	s service.Service
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
	var transaction common2.Transaction
	json.Unmarshal(resp.Body(), &transaction)
	ret := fmt.Sprintf("🏆 Leaderboard\n\n")
	return ret
}
