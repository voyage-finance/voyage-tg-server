package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/voyage-finance/voyage-tg-server/models"
	"github.com/voyage-finance/voyage-tg-server/service"
	"gorm.io/gorm"
	"log"
	"math/big"
	"os"
	"strconv"
	"strings"
)

type LlamaHandler struct {
	BaseHandler
	s            *service.Service
	erc20Handler *Erc20Handler
}

func NewLlamaHandler(s *service.Service, erc20Handler *Erc20Handler) *LlamaHandler {
	return &LlamaHandler{*NewBaseHandler("contracts/abis/llama.json"), s, erc20Handler}
}

func (llamaHandler *LlamaHandler) EncodeCreateStream(address string, amountPerSec int64) ([]byte, error) {
	methodName := "createStream"
	addressBytes := common.HexToAddress(address)
	amountPerSecBigInt := big.NewInt(amountPerSec)
	log.Println("amountPerSec", amountPerSec)
	if amountPerSecBigInt == big.NewInt(0) {
		return nil, errors.New("Provided amount for period duration is too small. Please increase amount or choose other duration!")
	}

	return llamaHandler.EncodeFunc(methodName, addressBytes, amountPerSecBigInt)
}

func (llamaHandler *LlamaHandler) EncodeDeposit(amount float64) ([]byte, error) {
	methodName := "deposit"
	amountBig := big.NewInt(int64(amount))

	return llamaHandler.EncodeFunc(methodName, amountBig)
}

func (llamaHandler *LlamaHandler) GetNetworkCreds(chain string) (string, string) {
	switch chain {
	case "matic":
		return "polygon", "Polygon"
	default:
		return "mainnet", "Ethereum"
	}
}

// Token Contract part

func (llamaHandler *LlamaHandler) GetLlamaTokenContract(chain string) *map[string]TokensType {
	networkChain, networkName := llamaHandler.GetNetworkCreds(chain)
	url := fmt.Sprintf("https://api.thegraph.com/subgraphs/name/nemusonaneko/llamapay-%v", networkChain)
	payload := ContractAddressRequest{
		Query: "\n    query GetAllTokens($network: String!) {\n  tokens(first: 500) {\n    address\n    symbol\n    name\n    decimals\n    contract {\n      id\n    }\n  }\n}\n    ",
		Variables: struct {
			Network string `json:"network"`
		}{
			Network: strings.ToTitle(networkName),
		},
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("GetLlamaTokenContract" + err.Error())
		return nil
	}

	resp, err := llamaHandler.s.Client.R().SetHeader("Content-Type", "application/json").SetBody(payloadJSON).Post(url)

	if err != nil {
		fmt.Println("GetLlamaTokenContract" + err.Error())
		return nil
	}
	var contractAddressResponse ContractAddressResponse
	err = json.Unmarshal(resp.Body(), &contractAddressResponse)
	if err != nil {
		fmt.Println("GetLlamaTokenContract" + err.Error())
		return nil
	}

	var symbolContractMap = map[string]TokensType{}
	for _, token := range contractAddressResponse.Data.Tokens {
		symbolContractMap[strings.ToLower(token.Address)] = token
	}
	return &symbolContractMap

}

// Token Contract part END

func (llamaHandler *LlamaHandler) getApprove(streamRequest *StreamRequest, value float64) MultiSignaturePayload {
	spender := streamRequest.TokenContract.Contract.Id // "0xA692FF8Fc672B513f7850C75465415437FE25617"

	erc20 := streamRequest.TokenContract.Address // "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174"
	approveData, _ := llamaHandler.erc20Handler.EncodeApprove(spender, int64(value))
	//approveDataString := hexutil.Encode(llamaHandler.EncodePacked(approveData, erc20, 0))
	return MultiSignaturePayload{
		To:   erc20,
		Data: hexutil.Encode(approveData),
	}
}

func (llamaHandler *LlamaHandler) getDeposit(streamRequest *StreamRequest, amountToDeposit float64) MultiSignaturePayload {
	llama := streamRequest.TokenContract.Contract.Id
	encodedDepositData, _ := llamaHandler.EncodeDeposit(amountToDeposit)
	//encodedDeposit := llamaHandler.EncodePacked(encodedDepositData, llama, 0)
	//encodedDepositString := hexutil.Encode(encodedDeposit)
	return MultiSignaturePayload{
		To:   llama,
		Data: hexutil.Encode(encodedDepositData),
	}
}

func (llamaHandler *LlamaHandler) getCreateStream(streamRequest *StreamRequest, amountPerSec float64) MultiSignaturePayload {
	llama := streamRequest.TokenContract.Contract.Id
	payee := streamRequest.Recipient.Address // "0xEB8fb2f6D41706759B8544D5adA16FC710211ca2"
	streamData, _ := llamaHandler.EncodeCreateStream(payee, int64(amountPerSec))
	//creatStream := llamaHandler.EncodePacked(streamData, streamRequest.TokenContract.Contract.Id, 0)
	//creatStreamString := hexutil.Encode(creatStream)
	return MultiSignaturePayload{
		To:   llama,
		Data: hexutil.Encode(streamData),
	}
}

func (llamaHandler *LlamaHandler) CreateStream(streamRequest *StreamRequest) []MultiSignaturePayload {
	result := []MultiSignaturePayload{}
	value := llamaHandler.s.SerializeBalance(fmt.Sprintf("%f", streamRequest.Amount), streamRequest.TokenContract.Decimals)
	valueStream := streamRequest.Amount * 10e20
	// approve
	approve := llamaHandler.getApprove(streamRequest, value)
	result = append(result, approve)
	// deposit
	deposit := llamaHandler.getDeposit(streamRequest, value)
	result = append(result, deposit)
	// create stream
	creatStream := llamaHandler.getCreateStream(streamRequest, valueStream/streamRequest.TotalSeconds)
	result = append(result, creatStream)

	//tns := append(approve, creatStream...)
	//return hexutil.Encode(tns)

	return result
}

func (llamaHandler *LlamaHandler) ValidateArgs(argString string, chat *models.Chat) (*StreamRequest, string) {
	args := strings.Fields(argString)
	// 1.0 check length of arguments
	if len(args) != 5 {
		return nil, "Must provide stream data in way: @recipient amount $currency period duration(day/month/year) ! E.g: /stream @user 1 $eth 1 month"
	}
	recipient, amount, currency, period, duration := args[0], args[1], args[2], args[3], args[4]
	// 2.0 recipient check, must start with @
	if recipient[0:1] != "@" {
		return nil, "Recipient should start with @! E.g: @user"
	}
	// 2.1 recipient should exist in Signer table
	recipient = strings.ToLower(recipient[1:])
	var recipientSigner models.Signer
	err := llamaHandler.s.DB.First(&recipientSigner, "chat_chat_id = ? AND name = ?", chat.ChatId, recipient).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Sprintf("*@%v* user does not exist in the chat", recipient)
	}

	// 3.0 currency check, should start from $
	if currency[0:1] != "$" {
		return nil, "Currency is not correct! E.g: $eth or $ETH"
	}

	currency = strings.ToLower(currency[1:])
	// 3.1 get currency contracts from subgraph
	currencyContracts := *llamaHandler.GetLlamaTokenContract(chat.Chain)
	if currencyContracts == nil {
		return nil, "Error in getting token contract!"
	}
	// 3.2 check currency
	chainId := 1
	if chat.Chain == "matic" {
		chainId = 137
	}
	url := fmt.Sprintf("https://safe-client.safe.global/v1/chains/%d/safes/%s/balances/usd", chainId, chat.SafeAddress)
	log.Println("request url: ", url)
	resp, err := llamaHandler.s.Client.R().EnableTrace().Get(url)
	if err != nil {
		log.Printf("get balance error: %s\n", err.Error())
	}

	type TokenInfo struct {
		Symbol  string
		Address string
	}

	type Item struct {
		TokenInfo TokenInfo
	}

	type BalanceRsp struct {
		Items []Item
	}

	var balanceRsp BalanceRsp
	json.Unmarshal(resp.Body(), &balanceRsp)

	var currencyAddr string
	var ok bool
	for _, item := range balanceRsp.Items {
		fmt.Printf("item.symbol: %s\n", item.TokenInfo.Symbol)
		if strings.ToLower(item.TokenInfo.Symbol) == strings.ToLower(currency) {
			ok = true
			currencyAddr = item.TokenInfo.Address
			break
		}
	}

	if !ok {
		return nil, "Balance is insufficient"
	}

	// 3.2 check whether currency supported for stream creation
	tokenContract, ok := currencyContracts[strings.ToLower(currencyAddr)]
	if !ok {
		return nil, "Contract is not supported!"
	}

	// 4.0 check period
	periodFloat, err := strconv.ParseFloat(period, 64)
	if err != nil {
		return nil, err.Error()
	}
	// 4.1 check duration
	var totalSeconds = float64(86400) // secs in 1 day
	if len(duration) < 3 {
		return nil, "Incorrect format for duration. Choose: day/month/year"
	}
	switch duration[:3] {
	case "day":
		totalSeconds *= periodFloat
	case "mon":
		totalSeconds *= periodFloat * float64(30)
	case "yea":
		totalSeconds *= periodFloat * float64(365)
	default:
		return nil, "Incorrect format for duration. Choose: day/month/year"
	}

	// 5.0 check amount
	amountFloat, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return nil, err.Error()
	}

	return &StreamRequest{
		Recipient:     recipientSigner,
		Amount:        amountFloat,
		Currency:      currency,
		TokenContract: tokenContract,
		TotalSeconds:  totalSeconds,
	}, ""
}

func (llamaHandler *LlamaHandler) getLink(chat *models.Chat, streamRequest *StreamRequest) string {
	txs := llamaHandler.CreateStream(streamRequest)

	var toData []string
	for _, payload := range txs {
		toData = append(toData, fmt.Sprintf("to=%s&data=%s", common.HexToAddress(payload.To), payload.Data))
	}

	baseUrl := fmt.Sprintf("%v/%v:%v/multisend?",
		os.Getenv("FRONT_URL"),
		chat.Chain,
		chat.SafeAddress,
	)

	url := baseUrl + strings.Join(toData, "&") + fmt.Sprintf("&chat_id=%v", chat.ChatId)
	return url
}

func (llamaHandler *LlamaHandler) Handle(chatId int64, args string) string {
	chat := llamaHandler.s.QueryChat(chatId)
	streamRequest, err := llamaHandler.ValidateArgs(args, chat)
	if err != "" {
		return err
	}
	link := llamaHandler.getLink(chat, streamRequest)
	if link != "" {
		return fmt.Sprintf("\U0001F999 Please sign stream creation transaction(s): approve, deposit, createStream\n\n"+
			"✍️ [Sign transaction(s)](%v)",
			link,
		)
	}
	return "Error in creation stream transaction(s)"

}
