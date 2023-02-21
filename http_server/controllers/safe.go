package controllers

// tutorial: https://dev.to/hackmamba/build-a-rest-api-with-golang-and-mongodb-gorillamux-version-57fh

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/voyage-finance/voyage-tg-server/contracts/handlers"
	"github.com/voyage-finance/voyage-tg-server/models"
	"github.com/voyage-finance/voyage-tg-server/service"
	"net/http"
)

type UserResponse struct {
	Name    string
	Address string
}

func GetSafeUsers(s service.Service) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// parse request body
		vars := mux.Vars(r)
		safeAddress := vars["safeAddress"]

		if safeAddress == "" {
			ReturnHttpBadResponse(rw, "Provide correct safe address in format 0x")
			return
		}

		var signers []models.Signer
		s.DB.Select([]string{"name", "address"}).
			Distinct().Joins("Chat").Where("safe_address = ?", safeAddress).
			Find(&signers)

		var usersResponse []UserResponse
		userResponseMap := map[string]string{}
		for _, user := range signers {
			addr := userResponseMap[user.Name]
			if addr != "" {
				continue
			}
			userResponseMap[user.Name] = user.Address
			usersResponse = append(usersResponse, UserResponse{
				Name:    user.Name,
				Address: user.Address,
			})
		}

		json.NewEncoder(rw).Encode(usersResponse)
	}
}

type CreateStreamSerializer struct {
	TokenContract handlers.TokensType `json:"tokenContract"`
	Amount        float64             `valid:"float"`
	Recipient     string              `json:"recipient"`
	Period        float64             `valid:"float"`
	Duration      string              `json:"duration"`
}

func GetEncodedApproveDepositCreateStream(s service.Service, llamaHandler *handlers.LlamaHandler) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var createStreamSerializer CreateStreamSerializer

		err := json.NewDecoder(r.Body).Decode(&createStreamSerializer)
		// 1.0 if body of request is not valid
		if err != nil {
			err := map[string]interface{}{"validationError": err}
			rw.Header().Set("Content-type", "application/json")
			json.NewEncoder(rw).Encode(err)
			return
		}

		totalSeconds, errorStr := llamaHandler.GetTotalSeconds(createStreamSerializer.Duration, createStreamSerializer.Period)
		if errorStr != "" {
			ReturnHttpBadResponse(rw, errorStr)
			return
		}

		recipient := models.Signer{
			Address: createStreamSerializer.Recipient,
		}

		streamRequest := &handlers.StreamRequest{
			Recipient:     recipient,
			Amount:        createStreamSerializer.Amount,
			Currency:      "",
			TokenContract: createStreamSerializer.TokenContract,
			TotalSeconds:  totalSeconds,
		}

		value := s.SerializeBalance(fmt.Sprintf("%f", streamRequest.Amount), streamRequest.TokenContract.Decimals)
		valueStream := streamRequest.Amount * 10e20

		var result []handlers.MultiSignaturePayload
		// approve
		approve := llamaHandler.GetApprove(streamRequest, value)
		result = append(result, approve)
		// deposit
		deposit := llamaHandler.GetDeposit(streamRequest, value)
		result = append(result, deposit)
		// create stream
		creatStream := llamaHandler.GetCreateStream(streamRequest, valueStream/streamRequest.TotalSeconds)
		result = append(result, creatStream)

		json.NewEncoder(rw).Encode(result)
	}
}
