package service

import (
	"encoding/json"
	"fmt"
)

func (s *Service) QueueTransaction(id int64) string {
	chat := s.QueryChat(id)
	r := fmt.Sprintf("https://safe-transaction-mainnet.safe.global/api/v1/safes/%s/all-transactions/?limit=1&executed=false&queued=true&trusted=true", chat.SafeAddress)
	resp, err := s.Client.R().EnableTrace().Get(r)
	if err != nil {
		return err.Error()
	}

	return resp.String()
}

func (s *Service) GenerateQueueLink(id int64) string {
	chat := s.QueryChat(id)
	return fmt.Sprintf("https://gnosis-safe.io/app/eth:%s/transactions/queue", chat.SafeAddress)
}

type StatusResp struct {
	Address   string   `json:"address"`
	Nonce     int64    `json:"nonce"`
	Threshold int64    `json:"threshold"`
	Owners    []string `jsom:"owners"`
}

func (s *Service) Status(id int64) string {
	chat := s.QueryChat(id)
	r := fmt.Sprintf("https://safe-transaction-mainnet.safe.global/api/v1/safes/%s/", chat.SafeAddress)
	resp, err := s.Client.R().EnableTrace().Get(r)
	if err != nil {
		return err.Error()
	}

	var statusResp StatusResp
	err = json.Unmarshal(resp.Body(), &statusResp)
	if err != nil {
		return err.Error()
	}

	return fmt.Sprintf(`Safe status
			Address: %s
			Nonce: %d
			Threshold: %d
			Owners: %+q

	`, statusResp.Address, statusResp.Nonce, statusResp.Threshold, statusResp.Owners)

}
