package routes

import (
	"github.com/gorilla/mux"
	"github.com/voyage-finance/voyage-tg-server/http_server/controllers"
	"gorm.io/gorm"
)

func OnboardingRoute(router *mux.Router, db *gorm.DB) {
	router.HandleFunc("/test", controllers.Test(db)).Methods("GET") //add this
}
