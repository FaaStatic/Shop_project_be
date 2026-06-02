package handler

import (
	"shop_project_be/internal/delivery/http/middleware"
	"shop_project_be/internal/domain"
	requestdto "shop_project_be/internal/dto/request_dto"
	"shop_project_be/pkg/response"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

type DebtHandler struct {
	usecase domain.DebtUseCase
	log     *zap.Logger
}

func NewDebtHandler(usecase domain.DebtUseCase, log *zap.Logger) *DebtHandler {
	return &DebtHandler{usecase: usecase, log: log}
}

// Add menangani POST /debts.
func (h *DebtHandler) Add(c fiber.Ctx) error {
	var req requestdto.AddDebtRequest
	if err := bindBody(c, &req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request body", err)
	}
	req.UserId = middleware.GetUserID(c)
	if err := validate.Validate(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation failed", err)
	}
	if err := h.usecase.AddingDebtCustomer(c.Context(), &req); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), err)
	}
	return response.Success(c, fiber.StatusCreated, "debt created", nil)
}

// Delete menangani DELETE /debts.
func (h *DebtHandler) Delete(c fiber.Ctx) error {
	var req requestdto.DeleteDebtRequest
	if err := bindBody(c, &req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request body", err)
	}
	req.UserId = middleware.GetUserID(c)
	if err := validate.Validate(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation failed", err)
	}
	if err := h.usecase.DeleteDebtCustomer(c.Context(), &req); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), err)
	}
	return response.Success(c, fiber.StatusOK, "debt deleted", nil)
}

// List menangani GET /debts.
func (h *DebtHandler) List(c fiber.Ctx) error {
	var req requestdto.FilterDebtRequest
	if err := bindQuery(c, &req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid query", err)
	}
	req.UserId = middleware.GetUserID(c)
	result, err := h.usecase.GetAllDebtCustomerList(c.Context(), &req)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), err)
	}
	return response.Success(c, fiber.StatusOK, "debts fetched", result)
}

// Get menangani GET /debts/:id.
func (h *DebtHandler) Get(c fiber.Ctx) error {
	req := requestdto.GetDebtRequest{DebtId: c.Params("id")}
	if err := validate.Validate(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation failed", err)
	}
	result, err := h.usecase.GetDebtCustomer(c.Context(), &req)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, err.Error(), err)
	}
	return response.Success(c, fiber.StatusOK, "debt found", result)
}

// Report menangani GET /debts/report (PDF laporan hutang customer).
func (h *DebtHandler) Report(c fiber.Ctx) error {
	req := requestdto.PrintDebtReport{
		UserId:       middleware.GetUserID(c),
		DebtId:       c.Query("debt_id"),
		NameCustomer: c.Query("name_customer"),
		Month:        c.Query("month"),
		Year:         c.Query("year"),
	}
	result, err := h.usecase.PrintReportDebtCustomer(c.Context(), &req)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), err)
	}
	return response.Success(c, fiber.StatusOK, "report generated", result)
}
