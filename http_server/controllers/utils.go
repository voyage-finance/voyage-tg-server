package controllers

import (
	"encoding/json"
	"log"
	"net/http"
)

func ReturnHttpBadResponse(rw http.ResponseWriter, response string) {
	rw.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(rw).Encode(response)
	log.Println(response)
}
