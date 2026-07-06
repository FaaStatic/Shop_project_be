package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"
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
		// A Bearer-token-based (stateless) API is not vulnerable to CSRF, so the
		// /auth and /api routes are skipped so they don't need a cookie+CSRF token.
		Next: func(c fiber.Ctx) bool {
			p := c.Path()
			return strings.HasPrefix(p, "/auth") || strings.HasPrefix(p, "/api")
		},
	}
}
