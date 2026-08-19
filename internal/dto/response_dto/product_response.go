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
	PurchasePrice    int64     `json:"purchase_price"`
	SellingPrice     int64     `json:"selling_price"`
	SellingPriceDebt int64     `json:"selling_price_debt"`
	Stock            float64   `json:"stock"`
	Category         string    `json:"category"`
	Image            string    `json:"image"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type ProductAddBulkResponse struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
}

type GetAllProductResponse struct {
	UserId      string               `json:"user_id"`
	NextId      string               `json:"next_id"`
	NextTime    string               `json:"next_time"`
	HasNext     bool                 `json:"has_next"`
	ProductList []ProductDtoResponse `json:"product_list"`
}
