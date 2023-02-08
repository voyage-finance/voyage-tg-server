package handlers

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type Erc20Handler struct {
	BaseHandler
}

func NewErc20Handler() *Erc20Handler {
	return &Erc20Handler{*NewBaseHandler("contracts/abis/erc20.json")}
}

func (erc20Handler *Erc20Handler) EncodeApprove(spender string, value int64) (string, error) {
	methodName := "approve"
	addressBytes := common.HexToAddress(spender)
	valueBigInt := big.NewInt(value)
	return erc20Handler.EncodeFunc(methodName, addressBytes, valueBigInt)
}
