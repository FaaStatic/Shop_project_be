// Package route mendaftarkan seluruh endpoint HTTP aplikasi. Fungsi New
// mengembalikan registrar yang dipanggil InitFiber sebelum handler not-found.
package route

import (
	"shop_project_be/internal/delivery/http/handler"
	"shop_project_be/internal/delivery/http/middleware"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"go.uber.org/zap"
)

// Handlers mengelompokkan seluruh handler agar mudah di-inject.
type Handlers struct {
	User        *handler.UserHandler
	Product     *handler.ProductHandler
	Transaction *handler.TransactionHandler
	Customer    *handler.CustomerHandler
	Debt        *handler.DebtHandler
}

// New membangun registrar route. Endpoint /auth bersifat publik; sisanya
// berada di bawah grup /api yang dilindungi JWT.
func New(h Handlers, jwtMw *middleware.JWTMiddleware, loginLimiterStore fiber.Storage, log *zap.Logger) func(router fiber.Router) {
	return func(router fiber.Router) {
		// Publik. Login dibatasi rate limit untuk meredam brute force.
		auth := router.Group("/auth")
		auth.Post("/login", limiter.New(middleware.GetLoginLimiter(loginLimiterStore)), h.User.Login)

		// Terproteksi JWT
		api := router.Group("/api", jwtMw.Auth(log))

		// Manajemen user: pembuatan akun hanya boleh oleh superadmin.
		// Superadmin pertama dibuat lewat command CLI `create-admin`.
		users := api.Group("/users")
		users.Post("", jwtMw.RequireRole("superadmin"), h.User.Register)
		// Logout boleh siapa saja yang sedang login.
		users.Post("/logout", h.User.Logout)
		// Daftar kasir online: hanya superadmin & admin (pemilik/pengawas toko).
		users.Get("/online", jwtMw.RequireRole("superadmin", "admin"), h.User.OnlineUsers)

		products := api.Group("/products")
		products.Post("", h.Product.Add)
		products.Post("/bulk", h.Product.AddBulk)
		products.Get("", h.Product.List)
		products.Get("/:id", h.Product.Get)
		products.Put("", h.Product.Update)
		products.Patch("/stock", h.Product.UpdateStock)
		products.Delete("", h.Product.Delete)

		transactions := api.Group("/transactions")
		transactions.Post("", h.Transaction.Add)
		transactions.Get("", h.Transaction.List)
		transactions.Get("/report/month", h.Transaction.ReportMonth)
		transactions.Get("/report/transaction", h.Transaction.ReportTransaction)
		transactions.Get("/:id", h.Transaction.Get)
		transactions.Delete("", h.Transaction.Delete)

		customers := api.Group("/customers")
		customers.Post("", h.Customer.Add)
		customers.Get("", h.Customer.List)
		customers.Get("/:id", h.Customer.Get)
		customers.Put("", h.Customer.Update)
		customers.Delete("", h.Customer.Delete)

		debts := api.Group("/debts")
		debts.Post("", h.Debt.Add)
		debts.Get("", h.Debt.List)
		debts.Get("/report", h.Debt.Report)
		debts.Get("/:id", h.Debt.Get)
		debts.Delete("", h.Debt.Delete)
	}
}
