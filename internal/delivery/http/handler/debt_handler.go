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

// Add godoc
//
//	@Summary		Tambah hutang customer
//	@Description	Mencatat hutang baru untuk seorang customer.
//	@Tags			Debts
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		requestdto.AddDebtRequest	true	"Data hutang"
//	@Success		201		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/api/debts [post]
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

// Delete godoc
//
//	@Summary		Hapus hutang
//	@Description	Menghapus catatan hutang berdasarkan ID.
//	@Tags			Debts
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		requestdto.DeleteDebtRequest	true	"ID hutang yang dihapus"
//	@Success		200		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/api/debts [delete]
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

// List godoc
//
//	@Summary		List hutang
//	@Description	Mengambil daftar hutang dengan filter customer dan periode.
//	@Tags			Debts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			customer_id	query		string	false	"Filter customer ID"
//	@Param			month		query		string	false	"Filter bulan"
//	@Param			year		query		string	false	"Filter tahun"
//	@Success		200			{object}	response.APIResponse
//	@Failure		400			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/api/debts [get]
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

// Get godoc
//
//	@Summary		Detail hutang
//	@Description	Mengambil detail hutang berdasarkan ID.
//	@Tags			Debts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Debt ID"
//	@Success		200	{object}	response.APIResponse
//	@Failure		400	{object}	response.APIResponse
//	@Failure		404	{object}	response.APIResponse
//	@Router			/api/debts/{id} [get]
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

// Report godoc
//
//	@Summary		Laporan hutang customer
//	@Description	Menghasilkan laporan PDF hutang customer berdasarkan filter.
//	@Tags			Debts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			debt_id			query		string	false	"Debt ID"
//	@Param			name_customer	query		string	false	"Nama customer"
//	@Param			month			query		string	false	"Bulan"
//	@Param			year			query		string	false	"Tahun"
//	@Success		200				{object}	response.APIResponse
//	@Failure		500				{object}	response.APIResponse
//	@Router			/api/debts/report [get]
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
