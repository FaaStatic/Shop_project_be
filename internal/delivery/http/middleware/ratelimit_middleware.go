package middleware

import (
	"strings"
	"time"

	"shop_project_be/pkg/response"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
)

// GetLoginLimiter limits login attempts: max 5 requests per 2 minutes per
// IP to reduce password brute-forcing. Exceeding the limit -> HTTP 429.
//
// store should ideally be Redis (see cache.NewLimiterStorage) so counts stay accurate
// when prefork is active. If store is nil, the limiter uses the built-in in-memory store
// (per process — less accurate in prefork mode).
func GetLoginLimiter(store fiber.Storage) limiter.Config {
	return limiter.Config{
		Max:        5,
		Expiration: 2 * time.Minute,
		Storage:    store,
		// Prefix required: all limiters share one Redis storage; without a prefix
		// the login/global/webhook counters for the same IP would mix together.
		KeyGenerator: func(c fiber.Ctx) string {
			return "login:" + c.IP()
		},
		LimiterMiddleware: limiter.SlidingWindow{},
		LimitReached: func(c fiber.Ctx) error {
			return response.Error(c, fiber.StatusTooManyRequests,
				"Too many request in Network please try again on 2 minutes!", nil)
		},
	}
}

// GetGlobalLimiter is general protection: max 50 requests per minute per IP.
// Use a store with a DIFFERENT PREFIX from the login limiter so counters do not
// collide.
func GetGlobalLimiter(store fiber.Storage) limiter.Config {
	return limiter.Config{
		Max:        50,
		Expiration: time.Minute,
		Storage:    store,
		KeyGenerator: func(c fiber.Ctx) string {
			return "global:" + c.IP()
		},
		LimitReached: func(c fiber.Ctx) error {
			return response.Error(c, fiber.StatusTooManyRequests,
				"Too many request in Network please try again on 2 minutes!", nil)
		},
		LimiterMiddleware: limiter.SlidingWindow{},
		Next: func(c fiber.Ctx) bool {
			p := c.Path()
			return p == "/" || strings.HasPrefix(p, "/storage") // swagger UI at "/" + static files
		},
	}
}

// GetWebhookLimiter protects the public webhook endpoint (Midtrans notifications)
// from flooding: 60 requests per minute per IP. Intentionally loose so legitimate retries
// from Midtrans are never rejected — real notifications are well below this limit.
func GetWebhookLimiter(store fiber.Storage) limiter.Config {
	return limiter.Config{
		Max:        60,
		Expiration: time.Minute,
		Storage:    store,
		KeyGenerator: func(c fiber.Ctx) string {
			return "webhook:" + c.IP()
		},
		LimiterMiddleware: limiter.SlidingWindow{},
		LimitReached: func(c fiber.Ctx) error {
			return response.Error(c, fiber.StatusTooManyRequests,
				"too many requests, please retry later", nil)
		},
	}
}
