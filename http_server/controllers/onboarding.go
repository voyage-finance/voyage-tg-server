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

		fmt.Println("publicKey: ", publicKey)
		fmt.Println("GetAddress: ", message.GetAddress())
		fmt.Println("GetStatement: ", message.GetStatement())

		// 3.0 check sign message in db
		var signMessage models.SignMessage
		err = s.DB.First(&signMessage, "id = ? AND message = ?", signedMsgSerializer.Id.Value, message.GetStatement()).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ReturnHttpBadResponse(rw, "Message not found!")
			return
		}

		if signMessage.IsVerified {
			json.NewEncoder(rw).Encode("Message already verified!")
			return
		}

		// 4.0 check user existence in db
		var user models.User
		err = s.DB.First(&user, signMessage.UserID).Error
		if err != nil {
			ReturnHttpBadResponse(rw, fmt.Sprintf("No user in system with id = %v", signMessage.UserID))
			return
		}

		// 5.0 check whether signing address exists in Safe UI
		owners := s.Status(signMessage.ChatID)                 // lowered in slice
		addr := strings.ToLower(message.GetAddress().String()) // lowered addr
		if !slices.Contains(owners, addr) {
			ReturnHttpBadResponse(rw, fmt.Sprintf("This is not owner %v", addr))
			return
		}

		response = s.AddSigner(signMessage.ChatID, user.UserName, addr)
		if response == "" {
			response = fmt.Sprintf("Added signer, address: %s", addr)
		}

		// 6.0 update signMessage
		signMessage.IsVerified = true
		s.DB.Save(&signMessage)

		json.NewEncoder(rw).Encode(response)

	}
}
