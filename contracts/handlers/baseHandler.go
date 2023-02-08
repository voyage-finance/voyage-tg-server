package handlers

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
	"log"
	"math/big"
	"os"
)

type BaseHandler struct {
	ABI abi.ABI
}

func NewBaseHandler(abiPath string) *BaseHandler {
	fileABI, err := os.Open(abiPath)
	if err != nil {
		log.Println("BaseHandler.NewBaseHandler error: " + err.Error())
		return nil
	}
	parsedABI, _ := abi.JSON(fileABI)
	return &BaseHandler{ABI: parsedABI}
}

func (baseHandler *BaseHandler) EncodeFunc(functionName string, args ...interface{}) (string, error) {
	encoded, err := baseHandler.ABI.Pack(functionName, args...)
	if err != nil {
		log.Println("baseHandler.EncodeFunc error: " + err.Error())
	}
	return hexutil.Encode(encoded), err
}

func (baseHandler *BaseHandler) EncodePacked(data string, toAddress string, valueInt uint64) string {
	// Concatenate the parameters and encode them using the abi.encodePacked method
	operation := uint8(0)
	to := common.HexToAddress(toAddress)
	value := new(big.Int).SetUint64(valueInt)
	dataLength := big.NewInt(int64(len(data)))
	encodedData := []interface{}{
		operation,
		to,
		value,
		dataLength,
		data,
	}

	packed, _ := rlp.EncodeToBytes(encodedData)

	fmt.Println("Packed data:", hexutil.Encode(packed))
	return hexutil.Encode(packed)
}
