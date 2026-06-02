package handler

import (
	"shop_project_be/internal/delivery/http/middleware"
	"shop_project_be/internal/domain"
	requestdto "shop_project_be/internal/dto/request_dto"
	"shop_project_be/pkg/response"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

type TransactionHandler struct {
	trxUsecase domain.TransactionUsecase
	log        *zap.Logger
}

func NewTransactionHandler(trxUsecase domain.TransactionUsecase, log *zap.Logger) *TransactionHandler {
	return &TransactionHandler{
		trxUsecase: trxUsecase,
		log:        log,
	}
}

// Add menangani POST /transactions.
func (h *TransactionHandler) Add(c fiber.Ctx) error {
	var req requestdto.AddTransactionRequest
	if err := bindBody(c, &req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request body", err)
	}
	req.UserId = middleware.GetUserID(c)
	if err := validate.Validate(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation failed", err)
	}
	if err := h.trxUsecase.AddTransaction(c.Context(), &req); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), err)
	}
	return response.Success(c, fiber.StatusCreated, "transaction created", nil)
}

// List menangani GET /transactions.
func (h *TransactionHandler) List(c fiber.Ctx) error {
	var req requestdto.FilterTransactionRequest
	if err := bindQuery(c, &req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid query", err)
	}
	req.UserId = middleware.GetUserID(c)
	result, err := h.trxUsecase.GetAllTransaction(c.Context(), &req)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), err)
	}
	return response.Success(c, fiber.StatusOK, "transactions fetched", result)
}

// Get menangani GET /transactions/:id.
func (h *TransactionHandler) Get(c fiber.Ctx) error {
	req := requestdto.GetTransactionRequest{
		ID:         c.Params("id"),
		UserId:     middleware.GetUserID(c),
		CustomerId: c.Query("customer_id"),
	}
	if err := validate.Validate(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation failed", err)
	}
	result, err := h.trxUsecase.GetTransaction(c.Context(), &req)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, err.Error(), err)
	}
	return response.Success(c, fiber.StatusOK, "transaction found", result)
}

// Delete menangani DELETE /transactions.
func (h *TransactionHandler) Delete(c fiber.Ctx) error {
	var req requestdto.DeleteTransactionRequest
	if err := bindBody(c, &req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request body", err)
	}
	if err := validate.Validate(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation failed", err)
	}
	if err := h.trxUsecase.DeleteTransaction(c.Context(), &req); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), err)
	}
	return response.Success(c, fiber.StatusOK, "transaction deleted", nil)
}

// ReportMonth menangani GET /transactions/report/month.
func (h *TransactionHandler) ReportMonth(c fiber.Ctx) error {
	var req requestdto.PrintReportMonthRequest
	if err := bindQuery(c, &req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid query", err)
	}
	req.UserId = middleware.GetUserID(c)
	if err := validate.Validate(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation failed", err)
	}
	result, err := h.trxUsecase.PrintReportMonth(c.Context(), &req)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), err)
	}
	return response.Success(c, fiber.StatusOK, "report generated", result)
}

// ReportTransaction menangani GET /transactions/report/transaction.
func (h *TransactionHandler) ReportTransaction(c fiber.Ctx) error {
	var req requestdto.PrintReportTransactionRequest
	if err := bindQuery(c, &req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid query", err)
	}
	req.UserId = middleware.GetUserID(c)
	if err := validate.Validate(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation failed", err)
	}
	result, err := h.trxUsecase.PrintReportTransaction(c.Context(), &req)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), err)
	}
	return response.Success(c, fiber.StatusOK, "report generated", result)
}
