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

// Add godoc
//
//	@Summary		Tambah produk
//	@Description	Menambahkan produk baru ke toko milik user yang login.
//	@Tags			Products
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		requestdto.AddProduct	true	"Data produk"
//	@Success		201		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/api/products [post]
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

// AddBulk godoc
//
//	@Summary		Import produk massal
//	@Description	Mengimpor banyak produk sekaligus dari file CSV/Excel.
//	@Tags			Products
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		BearerAuth
//	@Param			name_file	formData	string	false	"Nama file"
//	@Param			file_upload	formData	file	true	"File CSV/Excel produk"
//	@Success		201			{object}	response.APIResponse
//	@Failure		400			{object}	response.APIResponse
//	@Router			/api/products/bulk [post]
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

// Get godoc
//
//	@Summary		Detail produk
//	@Description	Mengambil detail produk berdasarkan ID.
//	@Tags			Products
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Product ID"
//	@Success		200	{object}	response.APIResponse
//	@Failure		400	{object}	response.APIResponse
//	@Failure		404	{object}	response.APIResponse
//	@Router			/api/products/{id} [get]
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

// List godoc
//
//	@Summary		List produk
//	@Description	Mengambil daftar produk milik user yang login, dengan filter dan pagination.
//	@Tags			Products
//	@Produce		json
//	@Security		BearerAuth
//	@Param			category	query		string	false	"Filter kategori"
//	@Param			search		query		string	false	"Pencarian nama/SKU produk"
//	@Param			page		query		int		false	"Halaman"
//	@Param			limit		query		int		false	"Jumlah data per halaman"
//	@Param			last_id		query		string	false	"ID terakhir untuk cursor pagination"
//	@Param			after_time	query		string	false	"Cursor waktu untuk pagination"
//	@Param			order		query		string	false	"Urutan data"
//	@Success		200			{object}	response.APIResponse
//	@Failure		400			{object}	response.APIResponse
//	@Failure		500			{object}	response.APIResponse
//	@Router			/api/products [get]
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

// Update godoc
//
//	@Summary		Update produk
//	@Description	Memperbarui atribut produk (tidak mengubah stok).
//	@Tags			Products
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		requestdto.UpdateProduct	true	"Data produk yang diperbarui"
//	@Success		200		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/api/products [put]
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

// UpdateStock godoc
//
//	@Summary		Update stok produk
//	@Description	Mengubah stok produk berdasarkan delta (positif menambah, negatif mengurangi).
//	@Tags			Products
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		requestdto.UpdateStock	true	"Delta stok produk"
//	@Success		200		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/api/products/stock [patch]
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

// Delete godoc
//
//	@Summary		Hapus produk
//	@Description	Menghapus produk berdasarkan ID.
//	@Tags			Products
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		requestdto.DeleteProduct	true	"ID produk yang dihapus"
//	@Success		200		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/api/products [delete]
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
