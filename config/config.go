package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
)

func Init() {
	// Getting configuration dir
	exPath, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	exPath += "/config/"

	botEnv, exists := os.LookupEnv("BOT_ENV")
	if !exists {
		botEnv = "dev"
	}
	envFile := fmt.Sprintf("%v.env.%v", exPath, botEnv)
	err = godotenv.Load(envFile+".local", envFile)

	if err != nil {
		log.Fatal("Error loading env files", err)
	}
	print("Loaded ENV variables")
}
