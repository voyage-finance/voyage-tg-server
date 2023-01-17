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

	bot_env, exists := os.LookupEnv("BOT_ENV")
	if !exists {
		bot_env = "dev"
	}
	err = godotenv.Load(exPath+".env."+bot_env+".local", exPath+".env."+bot_env)

	if err != nil {
		log.Fatal("Error loading env files", err)
	}
	print("Loaded ENV variables")
}
