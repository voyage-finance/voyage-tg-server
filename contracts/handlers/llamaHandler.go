package handlers

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type LlamaHandler struct {
	BaseHandler
}

func NewLlamaHandler() *LlamaHandler {
	return &LlamaHandler{*NewBaseHandler("contracts/abis/llama.json")}
}

func (llamaHandler *LlamaHandler) EncodeCreateStream(address string, amountPerSec int64) (string, error) {
	methodName := "createStream"
	addressBytes := common.HexToAddress(address)
	amountPerSecBigInt := big.NewInt(amountPerSec)

	return llamaHandler.EncodeFunc(methodName, addressBytes, amountPerSecBigInt)
}
