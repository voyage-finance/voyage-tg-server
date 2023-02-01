package common

type TransactionResponse struct {
	Next     int64 `json:"next"`
	Previous int64 `json:"previous"`
	Results  []struct {
		Type         string `json:"type"`
		ConflictType string `json:"conflictType"`
		Nonce        int64  `json:"nonce"`
		Transaction  `json:"transaction"`
	} `json:"results"`
}

type AddressValue struct {
	Value string
}

type AddressEx struct {
	Value   string
	Name    string
	LogoUri string `json:"logoUri"`
}

type ExecutionInfo struct {
	Type                   string
	Nonce                  int64
	ConfirmationsRequired  uint64 `json:"confirmationsRequired"`
	ConfirmationsSubmitted uint64 `json:"confirmationsSubmitted"`
	// DetailedExecutionInfoType.MODULE
	Address AddressValue `json:"address"`
}

type TxInfo struct {
	Type         string
	Sender       AddressValue
	Recipient    AddressValue
	Direction    string
	TransferInfo struct {
		Type  string
		Value string
		// Erc721Transfer
		TokenSymbol  string `json:"tokenSymbol"`
		TokenId      string `json:"tokenId"`
		TokenName    string `json:"tokenName"`
		LogoUri      string `json:"logoUri"`
		TokenAddress string `json:"tokenAddress"`
		// Erc20Transfer
		Decimals int32 `json:"decimals"`
	} `json:"transferInfo"`
	// custom type
	To             AddressEx `json:"to"`
	DataSize       string    `json:"dataSize"`
	Value          string    `json:"Value"`
	MethodName     string    `json:"methodName"`
	ActionCount    int       `json:"actionCount"`
	IsCancellation bool      `json:"isCancellation"`

	// settings change
	SettingsInfo struct {
		Type      string
		Threshold int
	} `json:"settingsInfo"`
}

type SafeAppInfo struct {
	Name    string
	Url     string
	LogoUri string `json:"logoUri"`
}

type Transaction struct {
	Id            string
	Timestamp     uint64
	TxStatus      string `json:"txStatus"`
	TxInfo        `json:"txInfo"`
	ExecutionInfo `json:"executionInfo"`
	SafeAppInfo   `json:"safeAppInfo"`
}

type RetrieveTransaction struct {
	Id            string `json:"txId"`
	Timestamp     uint64
	TxStatus      string `json:"txStatus"`
	TxInfo        `json:"txInfo"`
	ExecutionInfo `json:"detailedExecutionInfo"`
	SafeAppInfo   `json:"safeAppInfo"`
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
