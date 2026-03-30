package middleware

import (
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/gofiber/fiber/v3/middleware/encryptcookie"
	"github.com/joho/godotenv"
)

func GetSecureCookiesMiddleware() encryptcookie.Config {

	envData, err := godotenv.Read()
	if err != nil {
		panic(err)
	}
	var key string
	if envData["APP_ENV"] == "production" && envData["ENCRYPT_KEY"] != "" {
		key = envData["ENCRYPT_KEY"]
	} else {
		key = encryptcookie.GenerateKey(16)
	}

	return encryptcookie.Config{
		Key:    key,
		Except: []string{csrf.ConfigDefault.CookieName},
	}
}
