package service

import "fmt"

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
	Address   string
	Nonce     int64
	Threshold int64
	Owner     []string
}

func (s *Service) Status(id int64) string {
	chat := s.QueryChat(id)
	r := fmt.Sprintf("https://safe-transaction-mainnet.safe.global/api/v1/safes/%s/", chat.SafeAddress)
	resp, err := s.Client.R().EnableTrace().Get(r)
	if err != nil {
		return err.Error()
	}

	return string(resp.Body())
}
