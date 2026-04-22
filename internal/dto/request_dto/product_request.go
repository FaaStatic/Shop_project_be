package requestdto

import "mime/multipart"

type SearchProduct struct {
	ID          string `query:"id,omitempty"`
	Sku         string `query:"sku,omitempty"`
	ProductName string `query:"product_name,omitempty"`
}

type GetProduct struct {
	ID string `query:"id" validate:"required"`
}

type AddProduct struct {
	UserId           string  `json:"user_id" validate:"required"`
	SKU              string  `json:"sku" validate="required"`
	ProductName      string  `json:"product_name" validate="required"`
	Unit             int     `json:"unit,omitempty" validate:"omitempty,oneof=0 1 2 3 4"`
	PurchasePrice    float64 `json:"purchase_price" validate="required"`
	SellingPrice     float64 `json:"selling_price" validate="required"`
	SellingPriceDebt float64 `json:"selling_price_debt" validate="required"`
	Stock            int     `json:"stock" validate="required"`
	Category         string  `json:"category" validate="required"`
	Image            string  `json:"image,omitempty"`
}

type AddBulkProduct struct {
	UserId     string                `form:"user_id" validate:"required"`
	NameFile   string                `form:"name_file"`
	FileUpload *multipart.FileHeader `form:"file_upload"`
}

type DeleteProduct struct {
	ID uint `json:"id" validate:"required"`
}

type UpdateProduct struct {
	ID               string   `json:"id" validate:"required"`
	SKU              *string  `json:"sku,omitempty"`
	ProductName      *string  `json:"product_name,omitempty"`
	Unit             *int     `json:"unit,omitempty" validate:"omitempty,oneof=0 1 2 3 4"`
	PurchasePrice    *float64 `json:"purchase_price,omitempty" validate:"omitempty,gt=0"`
	SellingPrice     *float64 `json:"selling_price,omitempty" validate:"omitempty,gt=0"`
	SellingPriceDebt *float64 `json:"selling_price_debt,omitempty" validate:"omitempty,gte=0"`
	Stock            *int     `json:"stock,omitempty" validate:"omitempty,gte=0"`
	Category         *string  `json:"category,omitempty"`
	Image            *string  `json:"image,omitempty"`
}

type UpdateStock struct {
	ID    string `json:"id" validate:"required"`
	Stock int    `json:"stock,omitempty" validate:"required"`
}

type GetAllProduct struct {
	UserId string `json:"user_id" validate:"required"`
	Page   int    `query:"page"`
	Limit  int    `query:"limit"`
	Search string `query:"search"`
}
