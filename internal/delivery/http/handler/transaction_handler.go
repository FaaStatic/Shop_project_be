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

// Add godoc
//
//	@Summary		Tambah transaksi
//	@Description	Membuat transaksi penjualan baru beserta detail itemnya.
//	@Tags			Transactions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		requestdto.AddTransactionRequest	true	"Data transaksi"
//	@Success		201		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/api/transactions [post]
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

// List godoc
//
//	@Summary		List transaksi
//	@Description	Mengambil daftar transaksi milik user yang login dengan filter.
//	@Tags			Transactions
//	@Produce		json
//	@Security		BearerAuth
//	@Param			date_start		query		string	false	"Tanggal mulai (YYYY-MM-DD)"
//	@Param			date_end		query		string	false	"Tanggal akhir (YYYY-MM-DD)"
//	@Param			type_payment	query		int		true	"Tipe pembayaran (0=tunai,1=hutang,2=transfer,3=qris)"
//	@Param			number_invoices	query		string	false	"Filter nomor invoice"
//	@Param			after_time		query		string	false	"Cursor waktu untuk pagination"
//	@Param			after_id		query		string	false	"Cursor ID untuk pagination"
//	@Success		200				{object}	response.APIResponse
//	@Failure		400				{object}	response.APIResponse
//	@Failure		500				{object}	response.APIResponse
//	@Router			/api/transactions [get]
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

// Get godoc
//
//	@Summary		Detail transaksi
//	@Description	Mengambil detail transaksi berdasarkan ID.
//	@Tags			Transactions
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id			path		string	true	"Transaction ID"
//	@Param			customer_id	query		string	false	"Filter customer ID"
//	@Success		200			{object}	response.APIResponse
//	@Failure		400			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Router			/api/transactions/{id} [get]
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

// Delete godoc
//
//	@Summary		Hapus transaksi
//	@Description	Menghapus transaksi berdasarkan ID.
//	@Tags			Transactions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		requestdto.DeleteTransactionRequest	true	"ID transaksi yang dihapus"
//	@Success		200		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/api/transactions [delete]
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

// ReportMonth godoc
//
//	@Summary		Laporan transaksi bulanan
//	@Description	Menghasilkan laporan PDF rekap transaksi per bulan.
//	@Tags			Transactions
//	@Produce		json
//	@Security		BearerAuth
//	@Param			month	query		int	false	"Bulan (1-12)"
//	@Param			year	query		int	false	"Tahun"
//	@Success		200		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/api/transactions/report/month [get]
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

// ReportTransaction godoc
//
//	@Summary		Laporan detail transaksi
//	@Description	Menghasilkan laporan PDF untuk satu transaksi.
//	@Tags			Transactions
//	@Produce		json
//	@Security		BearerAuth
//	@Param			trx_id			query		string	false	"Transaction ID"
//	@Param			number_invoice	query		string	false	"Nomor invoice"
//	@Success		200				{object}	response.APIResponse
//	@Failure		400				{object}	response.APIResponse
//	@Failure		500				{object}	response.APIResponse
//	@Router			/api/transactions/report/transaction [get]
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
