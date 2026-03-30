package fiberconfig

import (
	"crypto/tls"
	loggerconfig "shop_project_be/pkg/logger"
	"time"

	middleware "shop_project_be/internal/delivery/middleware"

	"github.com/bytedance/sonic"
	swagger "github.com/gofiber/contrib/v3/swaggerui"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/gofiber/fiber/v3/middleware/encryptcookie"
	"github.com/gofiber/fiber/v3/middleware/helmet"
)

func GetFiberConfig() fiber.Config {
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

func InitFiber(env string) *fiber.App {
	app := fiber.New(GetFiberConfig())
	zapLogger := loggerconfig.LoggerCustom(env, app)
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
		app.Use(encryptcookie.New(middleware.GetSecureCookiesMiddleware()))
	}
	app.Use(middleware.LoggerMiddleware(zapLogger))
	app.Use(swagger.New(GetSwaggerConfig("Shop Project")))
	return app
}
