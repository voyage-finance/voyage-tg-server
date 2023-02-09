package handlers

import (
	"github.com/voyage-finance/voyage-tg-server/service"
)

type ContractHandlers struct {
	Erc20Handler *Erc20Handler
	LlamaHandler *LlamaHandler
	s            *service.Service
}

func NewContractHandlers(s *service.Service) *ContractHandlers {
	erc20Handler := NewErc20Handler()
	llamaHandler := NewLlamaHandler(s, erc20Handler)

	return &ContractHandlers{erc20Handler, llamaHandler, s}
}
