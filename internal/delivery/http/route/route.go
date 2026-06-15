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
func New(h Handlers, jwtMw *middleware.JWTMiddleware, storage fiber.Storage, log *zap.Logger) func(router fiber.Router) {
	return func(router fiber.Router) {
		// Publik. Register hanya membuat akun staff; admin/superadmin
		// dimasukkan langsung lewat DB.
		auth := router.Group("/auth")
		auth.Post("/login", limiter.New(middleware.GetLoginLimiter(storage)), h.User.Login)
		auth.Post("/register", limiter.New(middleware.GetLoginLimiter(storage)), h.User.Register)

		// Terproteksi JWT
		api := router.Group("/api", jwtMw.Auth(log))

		// onlySuper membatasi endpoint sensitif (delete, update produk,
		// report bulanan & hutang) hanya untuk superadmin.
		onlySuper := jwtMw.RequireRole("superadmin")

		products := api.Group("/products")
		products.Post("", h.Product.Add)
		products.Post("/bulk", h.Product.AddBulk)
		products.Get("", h.Product.List)
		products.Get("/:id", h.Product.Get)
		products.Put("", onlySuper, h.Product.Update)
		products.Patch("/stock", h.Product.UpdateStock)
		products.Delete("", onlySuper, h.Product.Delete)

		transactions := api.Group("/transactions")
		transactions.Post("", h.Transaction.Add)
		transactions.Get("", h.Transaction.List)
		transactions.Get("/report/month", onlySuper, h.Transaction.ReportMonth)
		transactions.Get("/report/transaction", h.Transaction.ReportTransaction)
		transactions.Get("/:id", h.Transaction.Get)
		transactions.Delete("", onlySuper, h.Transaction.Delete)

		customers := api.Group("/customers")
		customers.Post("", h.Customer.Add)
		customers.Get("", h.Customer.List)
		customers.Get("/:id", h.Customer.Get)
		customers.Put("", h.Customer.Update)
		customers.Delete("", onlySuper, h.Customer.Delete)

		debts := api.Group("/debts")
		debts.Post("", h.Debt.Add)
		debts.Get("", h.Debt.List)
		debts.Get("/report", onlySuper, h.Debt.Report)
		debts.Get("/:id", h.Debt.Get)
		debts.Delete("", onlySuper, h.Debt.Delete)
	}
}
