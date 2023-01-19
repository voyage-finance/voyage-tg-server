package service

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-resty/resty/v2"
	"gorm.io/gorm"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

type TokenInfo struct {
	TokenAddress string
	TokenType    string
	Name         string
	Symbol       string
	Decimals     int64
}

type Service struct {
	DB        *gorm.DB
	Client    *resty.Client
	EthClient *ethclient.Client
	Tokens    map[string]TokenInfo
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
	sig := signature

	message = accounts.TextHash(message)
	if sig[crypto.RecoveryIDOffset] == 27 || sig[crypto.RecoveryIDOffset] == 28 {
		sig[crypto.RecoveryIDOffset] -= 27 // Transform yellow paper V from 27/28 to 0/1
	}

	recovered, _ := crypto.SigToPub(message, sig)

	recoveredAddr := crypto.PubkeyToAddress(*recovered)
	return recoveredAddr.Hex()
}
