package models

type Chat struct {
	ChatId       int64  `gorm:"primaryKey;unique:true;not_null:true"`
	Title        string `json:"title"`
	Type         string `json:"type"`
	Chain        string `json:"chain"`
	SafeAddress  string `json:"safe_address"`
	Init         bool   `json:"init"`
	Signers      string `json:"signers"`
	SignMessages []SignMessage
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
