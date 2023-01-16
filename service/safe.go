package service

import (
	"encoding/json"
	"fmt"
)

type QueueTransactionResp struct {
	Count int64              `json:"count"`
	Next  string             `json:"next"`
	QT    []QueueTransaction `json:"results"`
}

type QueueTransaction struct {
	Safe           string
	To             string
	Value          string
	Data           string
	Operation      int64
	GasToken       string
	SafeTxGas      int64
	BaseGas        int64
	GasPrice       string
	RefundReceiver string
	Nonce          int64
	ExecutionDate  *string
	SubmissionDate *string
	Modified       *string
	SafeTxHash     *string
	DataDecoded    DataDecoded
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

func (p *Parameter) String() string {
	return fmt.Sprintf("[Name: %s, Type: %s, Value: %s]", p.Name, p.Type, p.Value)
}

func (s *Service) QueryTokenBalance(id int64) string {
	chat := s.QueryChat(id)
	r := fmt.Sprintf("https://safe-transaction-mainnet.safe.global/api/v1/safes/%s/balances/?trusted=false&exclude_spam=false", chat.SafeAddress)
	resp, err := s.Client.R().EnableTrace().Get(r)
	if err != nil {
		return err.Error()
	}
	var balances []TokenBalance
	json.Unmarshal(resp.Body(), &balances)
	ret := "Balances: "
	for _, balance := range balances {
		if len(balance.Token.Name) > 10 {
			continue
		}
		if balance.Token.Name == "" {
			balance.Token.Name = "ETH"
		}
		if balance.Token.Symbol == "" {
			balance.Token.Symbol = "ETH"
		}
		ret += fmt.Sprintf(`
		Name: %s
		Symbol: %s
		Decimas: %d
		Balance: %s
		`, balance.Token.Name, balance.Token.Symbol, balance.Token.Decimals, balance.Balance)
	}
	return ret

}

func (s *Service) QueueTransaction(id int64, limit int64) string {
	chat := s.QueryChat(id)
	r := fmt.Sprintf("https://safe-transaction-mainnet.safe.global/api/v1/safes/%s/all-transactions/?limit=%d&executed=false&queued=true&trusted=true", chat.SafeAddress, limit)
	resp, err := s.Client.R().EnableTrace().Get(r)
	if err != nil {
		return err.Error()
	}

	var queueTransactionResp QueueTransactionResp
	json.Unmarshal(resp.Body(), &queueTransactionResp)
	ret := "Pending Transactions: "
	for index, qt := range queueTransactionResp.QT {
		parameters := ""
		for _, p := range qt.DataDecoded.Parameters {
			parameters += p.String()
		}
		link := fmt.Sprintf("https://gnosis-safe.io/app/eth:%s/transactions/multisig_%s_%s", chat.SafeAddress, chat.SafeAddress, *qt.SafeTxHash)

		ret += fmt.Sprintf(`
			Index: %d
			Safe: %s
			To: %s
			Value: %s
			SubmissionDate: %s
			Modified: %s
			SafeTxHash: %s
			Method: %s
			Parameters: %s
			Sign/Submit it: %s
			`, index, qt.Safe, qt.To, qt.Value, *qt.SubmissionDate, *qt.Modified, *qt.SafeTxHash, qt.DataDecoded.Method, parameters, link)
	}

	return ret

}

func (s *Service) GenerateQueueLink(id int64) string {
	chat := s.QueryChat(id)
	return fmt.Sprintf("https://gnosis-safe.io/app/eth:%s/transactions/queue", chat.SafeAddress)
}

func (s *Service) GenerateHistoryLink(id int64) string {
	chat := s.QueryChat(id)
	return fmt.Sprintf("Track transaction history here: https://gnosis-safe.io/app/eth:%s/transactions/history", chat.SafeAddress)
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