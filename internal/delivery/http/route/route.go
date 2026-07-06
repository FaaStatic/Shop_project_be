// Package route registers all of the application's HTTP endpoints. The New function
// returns the registrar called by InitFiber before the not-found handler.
package route

import (
	"shop_project_be/internal/delivery/http/handler"
	"shop_project_be/internal/delivery/http/middleware"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"go.uber.org/zap"
)

// Handlers groups all handlers together for easy injection.
type Handlers struct {
	User        *handler.UserHandler
	Product     *handler.ProductHandler
	Transaction *handler.TransactionHandler
	Customer    *handler.CustomerHandler
	Debt        *handler.DebtHandler
	Payment     *handler.PaymentHandler
	Fcm         *handler.FcmHandler
}

// New builds the route registrar. The /auth endpoints are public; the rest
// are under the /api group protected by JWT.
func New(h Handlers, jwtMw *middleware.JWTMiddleware, storage fiber.Storage, log *zap.Logger) func(router fiber.Router) {
	return func(router fiber.Router) {
		// Public. Register only creates staff accounts; admin/superadmin
		// are inserted directly via the DB.
		auth := router.Group("/auth")
		auth.Post("/login", limiter.New(middleware.GetLoginLimiter(storage)), h.User.Login)
		auth.Post("/register", limiter.New(middleware.GetLoginLimiter(storage)), h.User.Register)

		// The Midtrans webhook is PUBLIC (no JWT). Its authenticity is validated
		// via signature_key in the usecase. Register this URL in the Midtrans dashboard.
		router.Post("/payments/notification", limiter.New(middleware.GetWebhookLimiter(storage)), h.Payment.Notification)

		// Terproteksi JWT
		api := router.Group("/api", jwtMw.Auth(log))

		// onlySuper restricts sensitive endpoints (delete, product update,
		// monthly & debt reports) only for superadmin.
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

		payments := api.Group("/payments")
		payments.Post("/qris", h.Payment.ChargeQris)
		payments.Post("/card", h.Payment.ChargeCard)
		payments.Get("/:order_id/status", h.Payment.Status)

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

		fcm := api.Group("/fcm")
		fcm.Post("/register", h.Fcm.Register)
		fcm.Post("/logout", h.Fcm.Logout)
	}
}
