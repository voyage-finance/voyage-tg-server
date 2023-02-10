package handlers

import "github.com/voyage-finance/voyage-tg-server/models"

type ContractAddressRequest struct {
	Query     string `json:"query"`
	Variables struct {
		Network string `json:"network"`
	} `json:"variables"`
}

type ContractAddressResponse struct {
	Data struct {
		Tokens []TokensType `json:"tokens"`
	} `json:"data"`
}

type TokensType struct {
	Address  string `json:"address"`
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Decimals int32  `json:"decimals"`
	Contract struct {
		Id string `json:"id"`
	} `json:"contract"`
}

type StreamRequest struct {
	Recipient     models.Signer
	Amount        float64
	Currency      string
	TokenContract TokensType
	TotalSeconds  float64
}

type MultiSignaturePayload struct {
	To   string `json:"to"`
	Data string `json:"data"`
}
