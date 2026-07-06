package fiberconfig

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"os"
	"time"

	envconfig "shop_project_be/config/env_config"
	middleware "shop_project_be/internal/delivery/http/middleware"
	appvalidator "shop_project_be/pkg/validator"

	"github.com/bytedance/sonic"
	swagger "github.com/gofiber/contrib/v3/swaggerui"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/gofiber/fiber/v3/middleware/encryptcookie"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/static"
	"go.uber.org/zap"
)

func GetFiberConfig(logger *zap.Logger, appName string) fiber.Config {
	return fiber.Config{
		JSONEncoder:     sonic.Marshal,
		JSONDecoder:     sonic.Unmarshal,
		StructValidator: appvalidator.New(),
		ServerHeader:    "Fiber",
		AppName:         appName,
		ReadTimeout:     5 * time.Minute,
		WriteTimeout:    5 * time.Minute,
		IdleTimeout:     10 * time.Minute,
		// 25 MB: enough for product CSV/Excel imports & normal JSON, but cuts
		// the DoS/RAM vector (larger requests are rejected 413 before being fully buffered).
		BodyLimit:     25 * 1024 * 1024,
		CaseSensitive: true,
		StrictRouting: true,
		ProxyHeader:   fiber.HeaderXForwardedFor,
		TrustProxy:    true,
		TrustProxyConfig: fiber.TrustProxyConfig{
			Proxies: []string{"10.0.0.0/8"},
			// If Nginx runs on the same host (localhost), this is enough:
			Loopback: true,
			// If the proxy is on a private network (e.g. docker network):
			Private: true,
		},
		ErrorHandler: func(c fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			message := "Something went wrong, please try again later"
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
				message = e.Message
			}
			logger.Error("Application is Crash",
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

// GetFiberConfigListener builds the listener configuration. gracefulCtx is used
// by Fiber to start a graceful shutdown when the context is cancelled (e.g. SIGINT/
// SIGTERM): the server stops accepting new connections then drains in-flight ones
// up to ShutdownTimeout. This only touches the start/stop lifecycle — it does not
// change any request processing or output.
func GetFiberConfigListener(env string, gracefulCtx context.Context) fiber.ListenConfig {
	return fiber.ListenConfig{
		EnablePrefork:     true,
		EnablePrintRoutes: env == "development",
		TLSMinVersion:     tls.VersionTLS12,
		GracefulContext:   gracefulCtx,
		ShutdownTimeout:   10 * time.Second,
	}
}

// loadSwaggerSpec reads the swagger.json produced by `make swagger` and overrides
// info.title with the app name from the yaml config, so the title
// shown in the Swagger UI also changes if server.name is changed.
func loadSwaggerSpec(nameApp string) []byte {
	raw, err := os.ReadFile("./swagger.json")
	if err != nil {
		return nil
	}

	var spec map[string]interface{}
	if err := json.Unmarshal(raw, &spec); err != nil {
		return raw
	}

	if info, ok := spec["info"].(map[string]interface{}); ok {
		info["title"] = nameApp + " API"
	}

	patched, err := json.Marshal(spec)
	if err != nil {
		return raw
	}
	return patched
}

func GetSwaggerConfig(nameApp string) swagger.Config {
	return swagger.Config{
		Next:        nil,
		BasePath:    "/",
		FilePath:    "./swagger.json",
		FileContent: loadSwaggerSpec(nameApp),
		Path:        "/",
		Title:       nameApp + " API documentation",
		CacheAge:    3600,
	}
}

// InitFiber builds the Fiber application along with its middleware.
// routes are the application's route-registrar functions; all are registered
// BEFORE the "not found" handler so the endpoints remain reachable.
func InitFiber(env string, envData *envconfig.Config, logger *zap.Logger, redisClient fiber.Storage, routes ...func(router fiber.Router)) *fiber.App {
	// Use the config already loaded & validated by the caller (envData); no
	// need to re-read the config file here. The values are identical, so output
	// is unchanged — it just removes one file read + duplicate validation.
	app := fiber.New(GetFiberConfig(logger, envData.App.Name))
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))
	app.Use(helmet.New(middleware.GetXSSConfig()))
	app.Use(compress.New(middleware.GetCompressConfig()))
	app.Use(cors.New(middleware.GetCorsConfig()))
	app.Use(csrf.New(middleware.GetCSRFConfig()))
	app.Use(limiter.New(middleware.GetGlobalLimiter(redisClient)))
	if env == "production" {
		app.Use(encryptcookie.New(middleware.GetSecureCookiesMiddleware(env, envData.Encrypt.Key)))
	}
	app.Use(middleware.LoggerMiddleware(logger))

	// Serve PDF report files as attachments so they download directly on the client.
	app.Use("/storage/reports", static.New("./storage/reports", static.Config{Download: true}))
	app.Use(swagger.New(GetSwaggerConfig(envData.App.Name)))

	// Register the application routes before the not-found handler.
	for _, register := range routes {
		register(app)
	}

	// The not-found handler must be last so it does not swallow other routes.
	app.Use(func(c fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Endpoint Not Found!",
		})
	})
	return app
}
