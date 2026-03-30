package envconfig

import (
	"log"

	"github.com/joho/godotenv"
)

func InitEnvConfig(env string) {
	if env == "production" {
		err := godotenv.Load(".env")
		if err != nil {
			log.Println("No .env file found")
		}
	} else {
		err := godotenv.Load(".env.development")
		if err != nil {
			log.Println("No .env file found")
		}
	}
}
