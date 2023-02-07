package http_server

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/voyage-finance/voyage-tg-server/http_server/controllers"
	"github.com/voyage-finance/voyage-tg-server/http_server/routes"
	"github.com/voyage-finance/voyage-tg-server/service"
	"log"
	"net/http"
	"os"
)

func HandleRequests(s service.Service) {
	bot := controllers.NewServerBot()
	// creates a new instance of a mux router
	router := mux.NewRouter().StrictSlash(true)
	// replace http.HandleFunc with myRouter.HandleFunc
	routes.OnboardingRoute(router, s, bot)
	routes.NotificationRoute(router, s, bot)

	// server run
	port := os.Getenv("HTTP_SERVER_PORT")
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", port), router))
	fmt.Printf("Bot-Server is running under the port: %v", port)
}
