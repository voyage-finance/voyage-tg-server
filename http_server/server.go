package http_server

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/voyage-finance/voyage-tg-server/http_server/routes"
	"github.com/voyage-finance/voyage-tg-server/service"
	"log"
	"net/http"
	"os"
)

func HandleRequests(s service.Service) {

	// creates a new instance of a mux router
	router := mux.NewRouter().StrictSlash(true)
	// replace http.HandleFunc with myRouter.HandleFunc
	routes.OnboardingRoute(router, s)

	// server run
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", os.Getenv("HTTP_SERVER_PORT")), router))
}
