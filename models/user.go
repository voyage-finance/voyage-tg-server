package models

import "gorm.io/gorm"

type User struct {
	UserId       int64 `gorm:"primaryKey;unique:true;not_null:true"`
	UserName     string
	SignMessages []SignMessage
	Chats        []*Chat `gorm:"many2many:signers;"`
}

type SignMessage struct {
	gorm.Model
	Message    string
	ChatID     int64
	UserID     int64
	IsVerified bool `gorm:"default:false"`
}
