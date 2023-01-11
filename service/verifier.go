package service

import (
	"crypto/sha256"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/go-resty/resty/v2"
	"gorm.io/gorm"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

type Service struct {
	DB     *gorm.DB
	Client *resty.Client
}

func (s *Service) GenerateMessage(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	t := time.Now().String()
	b = append(b, []byte(t)...)
	r := sha256.Sum256(b)
	return fmt.Sprintf("%x", r)
}

func (s *Service) RecoveryAddress(message []byte, signature []byte) string {
	sigPublicKey, err := crypto.Ecrecover(message, signature)
	if err != nil {
		log.Fatal(err)
	}
	unmarshalPubkey, _ := crypto.UnmarshalPubkey(sigPublicKey)
	return crypto.PubkeyToAddress(*unmarshalPubkey).String()
}
