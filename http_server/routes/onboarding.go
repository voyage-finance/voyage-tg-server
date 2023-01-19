package routes

import (
	"github.com/gorilla/mux"
	"github.com/voyage-finance/voyage-tg-server/http_server/controllers"
	"github.com/voyage-finance/voyage-tg-server/service"
)

func OnboardingRoute(router *mux.Router, s service.Service) {
	router.HandleFunc("/test", controllers.Test(s)).Methods("GET")             //add this
	router.HandleFunc("/verify", controllers.VerifyMessage(s)).Methods("POST") //add this
}
