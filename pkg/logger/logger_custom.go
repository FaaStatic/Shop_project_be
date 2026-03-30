package logger

import (
	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

func LoggerCustom(env string, app *fiber.App) *zap.Logger {
	if env == "production" {
		logger, _ := zap.NewProduction()
		defer logger.Sync()

		return logger
	} else {
		loggerDev, _ := zap.NewDevelopment()
		defer loggerDev.Sync()
		return loggerDev
	}
}
