package handler

import (
	"shop_project_be/internal/delivery/http/middleware"
	"shop_project_be/internal/domain"
	requestdto "shop_project_be/internal/dto/request_dto"
	"shop_project_be/pkg/response"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

type CustomerHandler struct {
	usecase domain.CustomerUsecase
	log     *zap.Logger
}

func NewCustomerHandler(usecase domain.CustomerUsecase, log *zap.Logger) *CustomerHandler {
	return &CustomerHandler{usecase: usecase, log: log}
}

// Add godoc
//
//	@Summary		Tambah customer
//	@Description	Menambahkan customer baru.
//	@Tags			Customers
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		requestdto.AddCustomer	true	"Data customer"
//	@Success		201		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/api/customers [post]
func (h *CustomerHandler) Add(c fiber.Ctx) error {
	var req requestdto.AddCustomer
	if err := bindBody(c, &req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request body", err)
	}
	req.UserId = middleware.GetUserID(c)
	if err := validate.Validate(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation failed", err)
	}
	if err := h.usecase.AddCustomerShop(c.Context(), &req); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), err)
	}
	return response.Success(c, fiber.StatusCreated, "customer created", nil)
}

// Update godoc
//
//	@Summary		Update customer
//	@Description	Memperbarui data customer.
//	@Tags			Customers
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		requestdto.UpdateCustomer	true	"Data customer yang diperbarui"
//	@Success		200		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/api/customers [put]
func (h *CustomerHandler) Update(c fiber.Ctx) error {
	var req requestdto.UpdateCustomer
	if err := bindBody(c, &req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request body", err)
	}
	req.UserId = middleware.GetUserID(c)
	if err := validate.Validate(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation failed", err)
	}
	if err := h.usecase.UpdateCustomerShop(c.Context(), &req); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), err)
	}
	return response.Success(c, fiber.StatusOK, "customer updated", nil)
}

// Delete godoc
//
//	@Summary		Hapus customer
//	@Description	Menghapus customer berdasarkan ID.
//	@Tags			Customers
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		requestdto.DeleteCustomer	true	"ID customer yang dihapus"
//	@Success		200		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/api/customers [delete]
func (h *CustomerHandler) Delete(c fiber.Ctx) error {
	var req requestdto.DeleteCustomer
	if err := bindBody(c, &req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request body", err)
	}
	req.UserId = middleware.GetUserID(c)
	if err := validate.Validate(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation failed", err)
	}
	if err := h.usecase.DeleteCustomerShop(c.Context(), &req); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), err)
	}
	return response.Success(c, fiber.StatusOK, "customer deleted", nil)
}

// Get godoc
//
//	@Summary		Detail customer
//	@Description	Mengambil detail customer berdasarkan ID.
//	@Tags			Customers
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Customer ID"
//	@Success		200	{object}	response.APIResponse
//	@Failure		400	{object}	response.APIResponse
//	@Failure		404	{object}	response.APIResponse
//	@Router			/api/customers/{id} [get]
func (h *CustomerHandler) Get(c fiber.Ctx) error {
	req := requestdto.GetCustomer{CustomerId: c.Params("id")}
	if err := validate.Validate(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation failed", err)
	}
	customer, err := h.usecase.GetCustomerShop(c.Context(), &req)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, err.Error(), err)
	}
	return response.Success(c, fiber.StatusOK, "customer found", customer)
}

// List godoc
//
//	@Summary		List customer
//	@Description	Mengambil daftar customer milik user yang login.
//	@Tags			Customers
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page	query		int		false	"Halaman"
//	@Param			limit	query		int		false	"Jumlah data per halaman"
//	@Param			search	query		string	false	"Pencarian nama customer"
//	@Success		200		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/api/customers [get]
func (h *CustomerHandler) List(c fiber.Ctx) error {
	var req requestdto.GetAllCustomer
	if err := bindQuery(c, &req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid query", err)
	}
	req.UserId = middleware.GetUserID(c)
	customers, err := h.usecase.GetListCustomerShop(c.Context(), &req)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), err)
	}
	return response.Success(c, fiber.StatusOK, "customers fetched", customers)
}
