package llama

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"log"
	"math/big"
	"os"
)

type LlamaHandler struct {
	ABI abi.ABI
}

func NewLlamaHandler() *LlamaHandler {
	fileABI, err := os.Open("contracts/abis/llama.json")
	if err != nil {
		log.Println("LlamaHandler.NewLlamaHandler error: " + err.Error())
		return nil
	}
	parsedABI, _ := abi.JSON(fileABI)
	return &LlamaHandler{ABI: parsedABI}
}

func (llamaHandler *LlamaHandler) EncodeFunc(functionName string, args ...interface{}) (string, error) {
	log.Println(args, "args")
	encoded, err := llamaHandler.ABI.Pack(functionName, args...)
	if err != nil {
		log.Println("LlamaHandler.EncodeFunc error: " + err.Error())
	}
	return hexutil.Encode(encoded), err
}

func (llamaHandler *LlamaHandler) EncodeCreateStream(address string, amountPerSec int64) (string, error) {
	methodName := "createStream"
	addressBytes := common.HexToAddress(address)
	amountPerSecBigInt := big.NewInt(amountPerSec)
	return llamaHandler.EncodeFunc(methodName, addressBytes, amountPerSecBigInt)
}
