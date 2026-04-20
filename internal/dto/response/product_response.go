package response

import "time"

type ProductDtoResponse struct {
	ID               uint      `json:"id"`
	SKU              string    `json:"sku"`
	ProductName      string    `json:"product_name"`
	Unit             int       `json:"unit"`
	PurchasePrice    float64   `json:"purchase_price"`
	SellingPrice     float64   `json:"selling_price"`
	SellingPriceDebt float64   `json:"selling_price_debt"`
	Stock            int       `json:"stock"`
	Category         string    `json:"category"`
	Image            string    `json:"image"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type ProductAddBulkResponse struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
}

type GetProductResponse struct {
	UserId   uint               `json:"user_id"`
	NameUser uint               `json:"name_user"`
	Product  ProductDtoResponse `json:"product"`
}

type GetAllProductResponse struct {
	UserId      uint                 `json:"user_id"`
	NameUser    uint                 `json:"name_user"`
	HashNext    bool                 `json:"hash_next"`
	NextCursor  uint                 `json:"next_cursor"`
	ProductList []ProductDtoResponse `json:"product_list"`
}
