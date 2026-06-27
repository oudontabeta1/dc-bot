package storage

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Env struct {
	DISCORD_BOT_TOKEN  string
	APPLICATION_ID     string
	TEST_GUILD_ID      string
	PANEL_API_TOKEN    string
	PANEL_CLIENT_TOKEN string
	OWNER_ID           string
	LOG_CHANNEL_ID     string
	IGNORE_GUILD_ID    string
	APP_ENV            string
}

var Envs = loadEnvs()

var ConfigMgr = NewConfigManager("config.json")

var (
	token          string
	application_id string
)

func loadEnvs() Env {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	env := os.Getenv("APP_ENV")
	if env == "" || env == "dev" || env == "development" {
		env = "development"
		application_id = os.Getenv("DEV_APPLICATION_ID")
		token = os.Getenv("DEV_DISCORD_BOT_TOKEN")
		log.Printf("Using Dev Enviroment")

	}
	if env == "production" || env == "prod" {
		env = "production"
		application_id = os.Getenv("PROD_APPLICATION_ID")
		token = os.Getenv("PROD_DISCORD_BOT_TOKEN")
		log.Printf("Using Prod Enviroment")

	}

	return Env{
		APPLICATION_ID:     application_id,
		DISCORD_BOT_TOKEN:  token,
		TEST_GUILD_ID:      os.Getenv("TEST_GUILD_ID"),
		PANEL_API_TOKEN:    os.Getenv("PANEL_API_TOKEN"),
		PANEL_CLIENT_TOKEN: os.Getenv("PANEL_CLIENT_TOKEN"),
		OWNER_ID:           os.Getenv("OWNER_ID"),
		LOG_CHANNEL_ID:     os.Getenv("LOG_CHANNEL_ID"),
		IGNORE_GUILD_ID:    os.Getenv("IGNORE_GUILD_ID"),
		APP_ENV:            os.Getenv("APP_ENV"),
	}
}
