package models

type Chat struct {
	ChatId      string `json:"chat_id"`
	Title       string `json:"title"`
	Type        string `json:"type"`
	SafeAddress string `json:"safe_address"`
	Init        bool   `json:"init"`
	Signers     string `json:"signers"`
}

type Signer struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

type PendingVerification struct {
	Message string `json:"message"`
	ChatId  string `json:"chat_id"`
	Name    string `json:"name"`
	Address string `json:"address"`
}
