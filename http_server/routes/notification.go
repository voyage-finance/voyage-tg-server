package routes

import (
	"github.com/gorilla/mux"
	"github.com/voyage-finance/voyage-tg-server/http_server/controllers"
	"github.com/voyage-finance/voyage-tg-server/service"
)

func NotificationRoute(router *mux.Router, s service.Service, serverBot *controllers.ServerBot) {
	router.HandleFunc("/notification/request", controllers.NotifyRequestSign(s, serverBot)).Methods("POST") //add this
}
