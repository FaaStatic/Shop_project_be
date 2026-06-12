package middleware

import "github.com/gofiber/fiber/v3/middleware/cors"

func GetCorsConfig() cors.Config {
	return cors.Config{
		// CATATAN KEAMANAN: "*" mengizinkan semua origin. Aman selama auth
		// memakai Bearer token (bukan cookie). Jika nanti pakai cookie auth,
		// ganti dengan daftar origin spesifik dan set AllowCredentials.
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
		// PATCH ditambahkan karena route PATCH /products/stock memerlukannya;
		// "UPDATE" dihapus karena bukan method HTTP yang valid.
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
	}
}
