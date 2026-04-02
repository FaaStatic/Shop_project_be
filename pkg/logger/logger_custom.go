package logger

import (
	"go.uber.org/zap"
)

func LoggerCustom(env string) *zap.Logger {
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
