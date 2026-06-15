package middleware

import (
	"strings"
	"time"

	"shop_project_be/pkg/response"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
)

// GetLoginLimiter membatasi percobaan login: maksimal 5 request per menit per
// IP untuk mengurangi brute force password. Melebihi batas -> HTTP 429.
//
// store sebaiknya Redis (lihat cache.NewLimiterStorage) agar hitungan akurat
// saat prefork aktif. Bila store nil, limiter pakai store in-memory bawaan
// (per proses — kurang akurat di mode prefork).
func GetLoginLimiter(store fiber.Storage) limiter.Config {
	return limiter.Config{
		Max:        5,
		Expiration: 2 * time.Minute,
		Storage:    store,
		KeyGenerator: func(c fiber.Ctx) string {
			return c.IP()
		},
		LimiterMiddleware: limiter.SlidingWindow{},
		LimitReached: func(c fiber.Ctx) error {
			return response.Error(c, fiber.StatusTooManyRequests,
				"Too many request in Network please try again on 2 minutes!", nil)
		},
	}
}

// GetGlobalLimiter adalah proteksi umum: maksimal 120 request per menit per IP.
// Pakai store dengan PREFIX BERBEDA dari login limiter agar counter tidak
// bertabrakan.
func GetGlobalLimiter(store fiber.Storage) limiter.Config {
	return limiter.Config{
		Max:        50,
		Expiration: time.Minute,
		Storage:    store,
		KeyGenerator: func(c fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c fiber.Ctx) error {
			return response.Error(c, fiber.StatusTooManyRequests,
				"Too many request in Network please try again on 2 minutes!", nil)
		},
		LimiterMiddleware: limiter.SlidingWindow{},
		Next: func(c fiber.Ctx) bool {
			p := c.Path()
			return p == "/" || strings.HasPrefix(p, "/storage") // swagger UI di "/" + file statis
		},
	}
}
