package middleware

import "github.com/gofiber/fiber/v3/middleware/cors"

func GetCorsConfig() cors.Config {
	return cors.Config{
		// SECURITY NOTE: "*" allows all origins. Safe as long as auth
		// memakai Bearer token (bukan cookie). Jika nanti pakai cookie auth,
		// replace it with a specific origin list and set AllowCredentials.
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
		// PATCH was added because the PATCH /products/stock route needs it;
		// "UPDATE" was removed because it is not a valid HTTP method.
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
	}
}
