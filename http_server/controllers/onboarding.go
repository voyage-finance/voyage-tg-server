package controllers

// tutorial: https://dev.to/hackmamba/build-a-rest-api-with-golang-and-mongodb-gorillamux-version-57fh

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/thedevsaddam/govalidator"
	"github.com/voyage-finance/voyage-tg-server/models"
	"github.com/voyage-finance/voyage-tg-server/service"
	"gorm.io/gorm"
	"net/http"
)

func Test(s service.Service) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var chats []models.Chat
		s.DB.Find(&chats)
		fmt.Println(chats, "------")
		json.NewEncoder(rw).Encode(chats)
	}
}

type SignedMessageSerializer struct {
	Id        govalidator.Int `json:"id"`
	Message   string          `json:"message"`
	Signature string          `json:"signature"`
}

func VerifyMessage(s service.Service) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// parse request body
		var signedMsgSerializer SignedMessageSerializer
		rules := govalidator.MapData{
			"id":        []string{"required"},
			"message":   []string{"required"},
			"signature": []string{"required"},
		}
		opts := govalidator.Options{
			Request: r,
			Data:    &signedMsgSerializer,
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

		// 2.0 check sign message in db
		var signMessage models.SignMessage
		err := s.DB.First(&signMessage, "id = ?", signedMsgSerializer.Id.Value).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			rw.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(rw).Encode("Message not found!")
			return
		}

		if signMessage.IsVerified {
			json.NewEncoder(rw).Encode("Message already verified!")
			return
		}

		// 3.0 check message by decoding
		message, err1 := hexutil.Decode(signedMsgSerializer.Message)
		response := ""
		if err1 != nil {
			response += " Wrong message"
		}
		// 3.1 check signature by decoding
		signature, err1 := hexutil.Decode(signedMsgSerializer.Signature)
		if err1 != nil {
			response += " Wrong signature"
		}

		if response != "" {
			rw.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(rw).Encode(response)
			return
		}

		addr := s.RecoveryAddress(message, signature)
		// 4.0 check user existence in db
		var user models.User
		err = s.DB.First(&user, signMessage.UserID).Error
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(rw).Encode(fmt.Sprintf("No user in system with id = %v", signMessage.UserID))
			return
		}

		// todo check if addr belongs to Status()
		ret := s.AddSigner(signMessage.ChatID, user.UserName, addr)
		if ret != "" {
			response = ret
		} else {
			response = fmt.Sprintf("Added signer, address: %s", addr)
		}

		signMessage.IsVerified = true
		s.DB.Save(&signMessage)

		json.NewEncoder(rw).Encode(response)

	}
}
