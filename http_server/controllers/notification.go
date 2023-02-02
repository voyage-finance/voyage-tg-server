package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/thedevsaddam/govalidator"
	"github.com/voyage-finance/voyage-tg-server/service"
	common2 "github.com/voyage-finance/voyage-tg-server/transaction/common"
	"github.com/voyage-finance/voyage-tg-server/transaction/queue"
	"net/http"
	"strconv"
)

type RequestNotificationSerializer struct {
	ChatId string `json:"chatId"`
	Chain  string `json:"chain"`
	TxId   string `json:"txId"`
}

func NotifyRequestSign(s service.Service, serverBot *ServerBot) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// parse request body
		var requestNotificationSerializer RequestNotificationSerializer
		rules := govalidator.MapData{
			"chatId": []string{"required"},
			"chain":  []string{"required"},
			"txId":   []string{"required"},
		}
		opts := govalidator.Options{
			Request: r,
			Data:    &requestNotificationSerializer,
			Rules:   rules,
		}
		parsedValue := govalidator.New(opts)
		e := parsedValue.ValidateJSON()
		// 1.0 if body of request is not valid
		if len(e) != 0 {
			err := map[string]interface{}{"validationError": e}
			rw.Header().Set("Content-type", "application/json")
			json.NewEncoder(rw).Encode(err)
			return
		}
		chatIdInt, err := strconv.ParseInt(requestNotificationSerializer.ChatId, 10, 64)
		if err != nil {
			json.NewEncoder(rw).Encode(err)
			return
		}

		chainId := s.GetChainId(requestNotificationSerializer.Chain)

		retrieveLink := fmt.Sprintf("https://safe-client.safe.global/v1/chains/%v/transactions/%v", chainId, requestNotificationSerializer.TxId)
		resp, err := s.Client.R().EnableTrace().Get(retrieveLink)
		if err != nil {
			json.NewEncoder(rw).Encode(err)
			return
		}
		var transactionRetrieve common2.RetrieveTransaction
		json.Unmarshal(resp.Body(), &transactionRetrieve)

		tx := common2.Transaction{
			Id:            transactionRetrieve.Id,
			Timestamp:     transactionRetrieve.Timestamp,
			TxStatus:      transactionRetrieve.TxStatus,
			TxInfo:        transactionRetrieve.TxInfo,
			ExecutionInfo: transactionRetrieve.ExecutionInfo,
			SafeAppInfo:   transactionRetrieve.SafeAppInfo,
		}

		queueHandler := queue.NewQueuedHandler(s)
		queueHandler.Setup(chatIdInt)
		response, link, isSupported := queueHandler.HandleTransaction(tx, false)

		if !isSupported {
			json.NewEncoder(rw).Encode("Error in Transaction parse")
			return
		}
		response = "🔔 *New Transaction Alert!* \n\n" + response

		isSent := serverBot.SendBotMessage(response, link, chatIdInt)

		json.NewEncoder(rw).Encode(isSent)

	}
}
