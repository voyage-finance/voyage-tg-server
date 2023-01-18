package http_server

import (
	"github.com/gorilla/mux"
	"github.com/voyage-finance/voyage-tg-server/http_server/routes"
	"gorm.io/gorm"
	"log"
	"net/http"
)

func HandleRequests(db *gorm.DB) {

	// creates a new instance of a mux router
	router := mux.NewRouter().StrictSlash(true)
	// replace http.HandleFunc with myRouter.HandleFunc
	routes.OnboardingRoute(router, db)

	// server run
	log.Fatal(http.ListenAndServe(":8070", router))
	//server := &http.Server{
	//	Addr: "0.0.0.0:8070",
	//	// Good practice to set timeouts to avoid Slowloris attacks.
	//	WriteTimeout: time.Second * 15,
	//	ReadTimeout:  time.Second * 15,
	//	IdleTimeout:  time.Second * 60,
	//	Handler:      myRouter, // Pass our instance of gorilla/mux in.
	//}
	//
	//// Run our server in a goroutine so that it doesn't block.
	//go func() {
	//	if err := server.ListenAndServe(); err != nil {
	//		log.Println(err)
	//	}
	//}()
	//
	//c := make(chan os.Signal, 1)
	//// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	//// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	//signal.Notify(c, os.Interrupt)
	//
	//// Block until we receive our signal.
	//<-c
	//
	//// Create a deadline to wait for.
	//ctx, cancel := context.WithTimeout(context.Background(), wait)
	//defer cancel()
	//// Doesn't block if no connections, but will otherwise wait
	//// until the timeout deadline.
	//server.Shutdown(ctx)
	//// Optionally, you could run srv.Shutdown in a goroutine and block on
	//// <-ctx.Done() if your application should wait for other services
	//// to finalize based on context cancellation.
	//log.Println("shutting down")
	//os.Exit(0)
}
