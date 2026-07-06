package responsedto

import (
	"time"

	"github.com/google/uuid"
)

type ProductDtoResponse struct {
	ID               uuid.UUID `json:"id"`
	SKU              string    `json:"sku"`
	ProductName      string    `json:"product_name"`
	Unit             int       `json:"unit"`
	PurchasePrice    float64   `json:"purchase_price"`
	SellingPrice     float64   `json:"selling_price"`
	SellingPriceDebt float64   `json:"selling_price_debt"`
	Stock            float64   `json:"stock"`
	Category         string    `json:"category"`
	Image            string    `json:"image"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type ProductAddBulkResponse struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
}

type GetProductResponse struct {
	UserId   string `json:"user_id"`
	NameUser string `json:"name_user"`

	Product []ProductDtoResponse `json:"product"`
}

type GetAllProductResponse struct {
	UserId      string               `json:"user_id"`
	NextId      string               `json:"next_id"`
	NextTime    string               `json:"next_time"`
	HasNext     bool                 `json:"has_next"`
	Page        int                  `json:"page"`
	ProductList []ProductDtoResponse `json:"product_list"`
}
