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
		// API berbasis Bearer token (stateless) tidak rentan CSRF, jadi route
		// /auth dan /api dilewati agar tidak butuh cookie+token CSRF.
		Next: func(c fiber.Ctx) bool {
			p := c.Path()
			return strings.HasPrefix(p, "/auth") || strings.HasPrefix(p, "/api")
		},
	}
}
