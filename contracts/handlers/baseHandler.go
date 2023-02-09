package handlers

import (
	"encoding/binary"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"log"
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

func (baseHandler *BaseHandler) EncodeFunc(functionName string, args ...interface{}) ([]byte, error) {
	encoded, err := baseHandler.ABI.Pack(functionName, args...)
	if err != nil {
		log.Println("baseHandler.EncodeFunc error: " + err.Error())
	}
	return encoded, err
}

func (baseHandler *BaseHandler) EncodePacked(data []byte, toAddress string, valueInt uint64) string {
	// Concatenate the parameters and encode them using the abi.encodePacked method
	op := []byte{0}
	to := common.HexToAddress(toAddress).Bytes()
	value := make([]byte, 32)
	fmt.Println("value:", value)
	binary.LittleEndian.PutUint64(value, uint64(0))
	dataLen := make([]byte, 32)
	binary.LittleEndian.PutUint64(dataLen, uint64(len(data)))
	fmt.Println("data len bytes: ", dataLen)
	packed := append(op, to...)
	packed = append(packed, dataLen...)
	packed = append(packed, data...)
	fmt.Println("Packed data:", hexutil.Encode(packed))
	return hexutil.Encode(packed)
}
