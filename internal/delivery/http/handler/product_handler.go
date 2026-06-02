package handler

import (
	"shop_project_be/internal/delivery/http/middleware"
	"shop_project_be/internal/domain"
	requestdto "shop_project_be/internal/dto/request_dto"
	"shop_project_be/pkg/response"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

type ProductHandler struct {
	usecase domain.ProductUsecase
	log     *zap.Logger
}

func NewProductHandler(usecase domain.ProductUsecase, log *zap.Logger) *ProductHandler {
	return &ProductHandler{usecase: usecase, log: log}
}

// Add menangani POST /products.
func (h *ProductHandler) Add(c fiber.Ctx) error {
	var req requestdto.AddProduct
	if err := bindBody(c, &req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request body", err)
	}
	req.UserId = middleware.GetUserID(c)
	if err := validate.Validate(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation failed", err)
	}
	if err := h.usecase.AddProductShopWithLock(c.Context(), &req); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), err)
	}
	return response.Success(c, fiber.StatusCreated, "product created", nil)
}

// AddBulk menangani POST /products/bulk (upload CSV/Excel).
func (h *ProductHandler) AddBulk(c fiber.Ctx) error {
	fileHeader, err := c.FormFile("file_upload")
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "file_upload is required", err)
	}
	req := requestdto.AddBulkProduct{
		UserId:     middleware.GetUserID(c),
		NameFile:   c.FormValue("name_file"),
		FileUpload: fileHeader,
	}
	if err := h.usecase.AddBulkProductShopWithLock(c.Context(), &req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), err)
	}
	return response.Success(c, fiber.StatusCreated, "bulk product imported", nil)
}

// Get menangani GET /products/:id.
func (h *ProductHandler) Get(c fiber.Ctx) error {
	req := requestdto.GetProduct{ID: c.Params("id")}
	if err := validate.Validate(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation failed", err)
	}
	product, err := h.usecase.GetProductShop(c.Context(), &req)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, err.Error(), err)
	}
	return response.Success(c, fiber.StatusOK, "product found", product)
}

// List menangani GET /products.
func (h *ProductHandler) List(c fiber.Ctx) error {
	var req requestdto.GetAllProduct
	if err := bindQuery(c, &req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid query", err)
	}
	req.UserId = middleware.GetUserID(c)
	products, err := h.usecase.GetAllProductShop(c.Context(), &req)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), err)
	}
	return response.Success(c, fiber.StatusOK, "products fetched", products)
}

// Update menangani PUT /products (atribut produk, tanpa mengubah stok).
func (h *ProductHandler) Update(c fiber.Ctx) error {
	var req requestdto.UpdateProduct
	if err := bindBody(c, &req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request body", err)
	}
	if err := validate.Validate(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation failed", err)
	}
	// Perubahan stok ditangani lewat endpoint stok khusus, jadi delta = 0.
	if err := h.usecase.UpdateProductShopWithLock(c.Context(), &req, 0); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), err)
	}
	return response.Success(c, fiber.StatusOK, "product updated", nil)
}

// UpdateStock menangani PATCH /products/stock. Field stock diperlakukan sebagai
// delta (positif menambah, negatif mengurangi) dan diterapkan dengan lock.
func (h *ProductHandler) UpdateStock(c fiber.Ctx) error {
	var req requestdto.UpdateStock
	if err := bindBody(c, &req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request body", err)
	}
	if err := validate.Validate(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation failed", err)
	}
	if err := h.usecase.UpdateStockWithLock(c.Context(), &req, req.Stock); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), err)
	}
	return response.Success(c, fiber.StatusOK, "stock updated", nil)
}

// Delete menangani DELETE /products.
func (h *ProductHandler) Delete(c fiber.Ctx) error {
	var req requestdto.DeleteProduct
	if err := bindBody(c, &req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request body", err)
	}
	if err := validate.Validate(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation failed", err)
	}
	if err := h.usecase.DeleteProductShop(c.Context(), &req); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), err)
	}
	return response.Success(c, fiber.StatusOK, "product deleted", nil)
}
