package middleware

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

func LoggerMiddleware(log *zap.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		// Catat waktu sebelum request diproses
		start := time.Now()

		// Lanjutkan ke handler/middleware berikutnya
		err := c.Next()

		// Hitung durasi request
		duration := time.Since(start)

		// Ambil status code HTTP
		statusCode := c.Response().StatusCode()

		// Gunakan Zap untuk nge-log data HTTP Request
		log.Info("HTTP Request",
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.Int("status", statusCode),
			zap.String("latency", duration.String()),
			zap.String("ip", c.IP()),
			zap.String("user_agent", string(c.Request().Header.UserAgent())),
		)

		return err
	}

}
