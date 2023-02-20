package routes

import (
	"github.com/gorilla/mux"
	"github.com/voyage-finance/voyage-tg-server/contracts/handlers"
	"github.com/voyage-finance/voyage-tg-server/http_server/controllers"
	"github.com/voyage-finance/voyage-tg-server/service"
)

func SafeRoute(router *mux.Router, s service.Service, serverBot *controllers.ServerBot, llamaHandler *handlers.LlamaHandler) {
	router.HandleFunc("/safe/users/{safeAddress}", controllers.GetSafeUsers(s)).Methods("GET")
	router.HandleFunc("/safe/encode", controllers.GetEncodedApproveDepositCreateStream(s, llamaHandler)).
		Methods("POST")
}
