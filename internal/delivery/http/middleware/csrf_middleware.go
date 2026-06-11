package middleware

import (
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/gofiber/fiber/v3/middleware/csrf"
)

func GetCSRFConfig() csrf.Config {
	return csrf.Config{
		CookieName:        "__Host-csrf_",
		CookieSecure:      true,
		CookieHTTPOnly:    true,
		CookieSameSite:    "Lax",
		CookieSessionOnly: true,
		Extractor:         extractors.FromHeader(csrf.HeaderName),
	}
}
