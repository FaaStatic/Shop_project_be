package middleware

import (
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/gofiber/fiber/v3/middleware/encryptcookie"
)

func GetSecureCookiesMiddleware(env string, encryptKey string) encryptcookie.Config {
	var key string
	if env == "production" && encryptKey != "" {
		key = encryptKey
	} else {
		key = encryptcookie.GenerateKey(16)
	}

	return encryptcookie.Config{
		Key:    key,
		Except: []string{csrf.ConfigDefault.CookieName},
	}
}
