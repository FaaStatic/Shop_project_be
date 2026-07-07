package middleware

import (
	"shop_project_be/pkg/response"

	"github.com/gofiber/fiber/v3"
)

// RequireFeature blocks a route group when the backing integration isn't
// configured (e.g. Midtrans keys absent), returning a clean 503 instead of
// letting the handler run against a half-configured gateway. This keeps
// missing optional config from crashing the whole app at startup while
// still making the gated routes visibly unusable.
func RequireFeature(enabled bool, feature string) fiber.Handler {
	return func(c fiber.Ctx) error {
		if !enabled {
			return response.Error(c, fiber.StatusServiceUnavailable, feature+" is not configured on this server", nil)
		}
		return c.Next()
	}
}
