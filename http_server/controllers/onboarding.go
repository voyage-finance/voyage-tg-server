package controllers

// tutorial: https://dev.to/hackmamba/build-a-rest-api-with-golang-and-mongodb-gorillamux-version-57fh

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spruceid/siwe-go"
	"github.com/thedevsaddam/govalidator"
	"github.com/voyage-finance/voyage-tg-server/models"
	"github.com/voyage-finance/voyage-tg-server/service"
	"golang.org/x/exp/slices"
	"gorm.io/gorm"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const domain = "example.com"
const addressStr = "0x71C7656EC7ab88b098defB751B7401B5f6d8976F"

var address = common.HexToAddress(addressStr)

const uri = "https://example.com"
const version = "1"
const statement = "Example statement for SIWE"

var issuedAt = time.Now().UTC().Format(time.RFC3339)
var nonce = siwe.GenerateNonce()

const chainId = 1

var expirationTime = time.Now().UTC().Add(48 * time.Hour).Format(time.RFC3339)

var notBefore = time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)

const requestId = "some-id"

var resourcesStr = []string{"https://example.com/resources/1", "https://example.com/resources/2"}

func parsedResources() []url.URL {
	parsed := make([]url.URL, len(resourcesStr))
	for i, resource := range resourcesStr {
		url, _ := url.Parse(resource)
		parsed[i] = *url
	}
	return parsed
}

var resources = parsedResources()

var options = map[string]interface{}{
	"statement":      statement,
	"version":        version,
	"chainId":        chainId,
	"issuedAt":       issuedAt,
	"expirationTime": expirationTime,
	"notBefore":      notBefore,
	"requestId":      requestId,
	"resources":      resources,
}

var message, _ = siwe.InitMessage(
	domain,
	addressStr,
	uri,
	nonce,
	options,
)

func Test(s service.Service) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		json.NewEncoder(rw).Encode("Git actions is working")
	}
}

type SignedMessageSerializer struct {
	Id        govalidator.Int `json:"id"`
	Message   string          `json:"message"`
	Signature string          `json:"signature"`
}

type LinkSafeSerializer struct {
	Id          govalidator.Int `json:"id"`
	Message     string          `json:"message"`
	Signature   string          `json:"signature"`
	SafeAddress string          `json:"safeAddress"`
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

		// 2.0 validate message
		message, err := siwe.ParseMessage(signedMsgSerializer.Message)
		response := ""
		if err != nil {
			response = fmt.Sprintf("message error: %v \n", err)
			ReturnHttpBadResponse(rw, response)
			return
		}
		publicKey, err := message.VerifyEIP191(signedMsgSerializer.Signature)
		if err != nil {
			response = fmt.Sprintf("signature error: %v \n", err)
			ReturnHttpBadResponse(rw, response)
			return
		}

		log.Println("publicKey: ", publicKey)
		log.Println("GetAddress: ", message.GetAddress())
		log.Println("GetStatement: ", message.GetStatement())

		// 3.0 check sign message in db
		var signMessage models.SignMessage
		err = s.DB.First(&signMessage, "id = ? AND message = ?", signedMsgSerializer.Id.Value, message.GetStatement()).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ReturnHttpBadResponse(rw, "Message not found!")
			return
		}

		// 4.0 check whether signing address exists in Safe UI
		addr := strings.ToLower(message.GetAddress().String()) // lowered addr

		response = s.AddSigner(signMessage.ChatID, signMessage.UserID, addr)
		if response == "" {
			response = fmt.Sprintf("Added signer, address: %s", addr)
		}

		// 5.0 update signMessage
		signMessage.IsVerified = true
		s.DB.Save(&signMessage)

		json.NewEncoder(rw).Encode(response)

	}
}

func LinkSafe(s service.Service, serverBot *ServerBot) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		// parse request body
		var linkSafeSerializer LinkSafeSerializer
		rules := govalidator.MapData{
			"id":          []string{"required"},
			"message":     []string{"required"},
			"signature":   []string{"required"},
			"safeAddress": []string{"required"},
		}
		opts := govalidator.Options{
			Request: r,
			Data:    &linkSafeSerializer,
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

		// 2.0 validate message
		message, err := siwe.ParseMessage(linkSafeSerializer.Message)
		response := ""
		if err != nil {
			response = fmt.Sprintf("message error: %v \n", err)
			ReturnHttpBadResponse(rw, response)
			return
		}
		publicKey, err := message.VerifyEIP191(linkSafeSerializer.Signature)
		if err != nil {
			response = fmt.Sprintf("signature error: %v \n", err)
			ReturnHttpBadResponse(rw, response)
			return
		}

		log.Println("publicKey: ", publicKey)
		log.Println("GetAddress: ", message.GetAddress())
		log.Println("GetStatement: ", message.GetStatement())

		// 3.0 check sign message in db
		var signMessage models.SignMessage
		err = s.DB.First(&signMessage, "id = ? AND message = ?", linkSafeSerializer.Id.Value, message.GetStatement()).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ReturnHttpBadResponse(rw, "Message not found!")
			return
		}

		// No need to check verification of SignMessage in Link logic

		// validate chain
		chainId := message.GetChainID()
		chain := s.GetSafeChain(chainId)

		// 4.0 save new safeAddress
		addr := strings.ToLower(message.GetAddress().String()) // lowered addr
		chat := s.QueryChat(signMessage.ChatID)
		// 4.1 update chat if safeAddress is updated
		if chat.SafeAddress != linkSafeSerializer.SafeAddress {
			chat.SafeAddress = linkSafeSerializer.SafeAddress
			chat.Chain = chain
			s.DB.Save(chat)
		}
		// 4.2 have up-to-date chain info
		if chat.Chain != chain {
			chat.Chain = chain
			s.DB.Save(chat)
		}

		// 4.3 check whether signing address exists in Safe UI
		owners := s.Status(signMessage.ChatID) // lowered in slice
		if !slices.Contains(owners, addr) {
			ReturnHttpBadResponse(rw, fmt.Sprintf("This is not owner %v", addr))
			return
		}

		//one_time_scripts.UpdateChatSignersOwnership(s, *chat)
		response = s.AddSigner(signMessage.ChatID, signMessage.UserID, addr)
		if response == "" {
			response = fmt.Sprintf("Added signer, address: %s", addr)
		}

		msg := fmt.Sprintf("The Safe=`%v` was set in this chat. Please send /link to bind personal address or send /help to know further instructions", addr)
		tgMessage := ConstructSignupMessage(msg, chat.ChatId)
		serverBot.SendBotMessage(tgMessage)

		// 5.0 update signMessage
		signMessage.IsVerified = true
		s.DB.Save(&signMessage)

		json.NewEncoder(rw).Encode(response)
	}
}
