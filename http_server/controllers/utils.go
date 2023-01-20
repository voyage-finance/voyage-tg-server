package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func ReturnHttpBadResponse(rw http.ResponseWriter, response string) {
	rw.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(rw).Encode(response)
	fmt.Println(response)
}
