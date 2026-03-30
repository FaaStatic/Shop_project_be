package main

import (
	"os"
	configdb "shop_project_be/internal/config/config_db"
	envconfig "shop_project_be/internal/config/env_config"
	fiberconfig "shop_project_be/internal/config/fiber_config"

	"github.com/gofiber/fiber/v3/log"
	"github.com/joho/godotenv"
)

func main() {
	env := os.Getenv("ENV")
	envconfig.InitEnvConfig(env)
	app := fiberconfig.InitFiber(env)
	configdb.InitDB()
	envApp, err := godotenv.Read()
	if err != nil {
		log.Fatal("Failed Load .env file!")
	}
	log.Info("Starting " + envApp["APP_NAME"] + " API on port " + envApp["APP_PORT"])
	log.Fatal(app.Listen(":"+envApp["APP_PORT"], fiberconfig.GetFiberConfigListener(env)))
}
