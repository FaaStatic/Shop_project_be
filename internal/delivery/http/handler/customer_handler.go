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

// Add menangani POST /customers.
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

// Update menangani PUT /customers.
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

// Delete menangani DELETE /customers.
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

// Get menangani GET /customers/:id.
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

// List menangani GET /customers.
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
