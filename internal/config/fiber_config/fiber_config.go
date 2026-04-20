package fiberconfig

import (
	"crypto/tls"
	"time"

	envconfig "shop_project_be/internal/config/env_config"
	middleware "shop_project_be/internal/delivery/middleware"

	"github.com/bytedance/sonic"
	swagger "github.com/gofiber/contrib/v3/swaggerui"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/gofiber/fiber/v3/middleware/encryptcookie"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"go.uber.org/zap"
)

func GetFiberConfig(logger *zap.Logger) fiber.Config {
	return fiber.Config{
		JSONEncoder:   sonic.Marshal,
		JSONDecoder:   sonic.Unmarshal,
		ServerHeader:  "Fiber",
		AppName:       "Shop Project BE",
		ReadTimeout:   5 * time.Second,
		WriteTimeout:  10 * time.Second,
		IdleTimeout:   120 * time.Second,
		BodyLimit:     1024 * 1024 * 1024,
		CaseSensitive: true,
		StrictRouting: true,
		ErrorHandler: func(c fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			message := "something went wrong, please try again later"
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
				message = e.Message
			}
			logger.Error("Aplikasi mengalami error atau panic",
				zap.Error(err),
				zap.Int("status_code", code),
				zap.String("path", c.Path()),
				zap.String("method", c.Method()),
				zap.String("ip", c.IP()),
			)

			return c.Status(code).JSON(fiber.Map{
				"success": false,
				"code":    code,
				"message": message,
			})
		},
	}
}

func GetFiberConfigListener(env string) fiber.ListenConfig {
	return fiber.ListenConfig{
		EnablePrefork:     true,
		EnablePrintRoutes: env == "development",
		TLSMinVersion:     tls.VersionTLS10,
	}
}

func GetSwaggerConfig(nameApp string) swagger.Config {
	return swagger.Config{
		Next:     nil,
		BasePath: "/",
		FilePath: "./swagger.json",
		Path:     "docs",
		Title:    nameApp + " API documentation",
		CacheAge: 3600,
	}
}

func InitFiber(env string, envData *envconfig.Config, logger *zap.Logger) *fiber.App {
	app := fiber.New(GetFiberConfig(logger))
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))
	app.Use(helmet.New(middleware.GetXSSConfig()))
	app.Use(compress.New(middleware.GetCompressConfig()))
	app.Use(cors.New(middleware.GetCorsConfig()))
	app.Use(csrf.New(middleware.GetCSRFConfig()))
	app.Use(func(c fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Endpoint Not Found!",
		})
	})
	if env == "production" {
		app.Use(encryptcookie.New(middleware.GetSecureCookiesMiddleware(env, envData.Encrypt.Key)))
	}
	app.Use(middleware.LoggerMiddleware(logger))
	app.Use(swagger.New(GetSwaggerConfig("Shop Project")))
	return app
}
