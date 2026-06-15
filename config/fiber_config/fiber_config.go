package fiberconfig

import (
	"crypto/tls"
	"encoding/json"
	"os"
	"strings"
	"time"

	envconfig "shop_project_be/config/env_config"
	middleware "shop_project_be/internal/delivery/http/middleware"
	"shop_project_be/pkg/response"
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
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    10 * time.Second,
		IdleTimeout:     120 * time.Second,
		BodyLimit:       1024 * 1024 * 1024,
		CaseSensitive:   true,
		StrictRouting:   true,
		ProxyHeader:     fiber.HeaderXForwardedFor,
		TrustProxy:      true,
		TrustProxyConfig: fiber.TrustProxyConfig{
			Proxies: []string{"10.0.0.0/8"},
			// Bila Nginx jalan di host yang sama (localhost), cukup:
			Loopback: true,
			// Bila proxy di jaringan privat (mis. docker network):
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

func GetFiberConfigListener(env string) fiber.ListenConfig {
	return fiber.ListenConfig{
		EnablePrefork:     true,
		EnablePrintRoutes: env == "development",
		TLSMinVersion:     tls.VersionTLS12,
	}
}

// loadSwaggerSpec membaca swagger.json hasil `make swagger` dan menimpa
// info.title dengan nama aplikasi dari config yaml, supaya judul yang
// tampil di Swagger UI ikut berubah jika server.name diganti.
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

// InitFiber membangun aplikasi Fiber beserta middleware-nya.
// routes adalah fungsi-fungsi pendaftar route aplikasi; semuanya didaftarkan
// SEBELUM handler "not found" agar endpoint tetap terjangkau.
func InitFiber(env string, envData *envconfig.Config, logger *zap.Logger, redisClient fiber.Storage, routes ...func(router fiber.Router)) *fiber.App {
	envConf, err := envconfig.InitEnvConfig(logger)
	if err != nil {
		logger.Fatal("Failed to initialize environment configuration", zap.Error(err))
	}
	app := fiber.New(GetFiberConfig(logger, envConf.App.Name))
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))
	app.Use(helmet.New(middleware.GetXSSConfig()))
	app.Use(compress.New(middleware.GetCompressConfig()))
	app.Use(cors.New(middleware.GetCorsConfig()))
	app.Use(csrf.New(middleware.GetCSRFConfig()))
	app.Use(limiter.New(limiter.Config{
		Max:        5,
		Expiration: 1 * time.Minute,
		Storage:    redisClient,
		LimitReached: func(c fiber.Ctx) error {
			return response.Error(c, fiber.StatusTooManyRequests,
				"too many request in Network please try again on 2 minutes!", nil)
		},
		KeyGenerator:      func(c fiber.Ctx) string { return c.IP() },
		LimiterMiddleware: limiter.SlidingWindow{},
		Next: func(c fiber.Ctx) bool {
			p := c.Path()
			return p == "/" || strings.HasPrefix(p, "/storage") // swagger UI di "/" + file statis
		},
	}))
	if env == "production" {
		app.Use(encryptcookie.New(middleware.GetSecureCookiesMiddleware(env, envData.Encrypt.Key)))
	}
	app.Use(middleware.LoggerMiddleware(logger))

	// Serve file laporan PDF sebagai attachment agar langsung ter-download di client.
	app.Use("/storage/reports", static.New("./storage/reports", static.Config{Download: true}))
	app.Use(swagger.New(GetSwaggerConfig(envConf.App.Name)))

	// Daftarkan route aplikasi sebelum handler not-found.
	for _, register := range routes {
		register(app)
	}

	// Handler not-found harus paling akhir agar tidak menelan route lain.
	app.Use(func(c fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Endpoint Not Found!",
		})
	})
	return app
}
