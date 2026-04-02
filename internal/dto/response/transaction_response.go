package response

type AddTransactionResponse struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
}

type GetProductTransactionResponse struct {
	ProductID   uint    `json:"product_id"`
	Quantity    uint    `json:"quantity"`
	NamaProduct string  `json:"nama_product"`
	Price       float64 `json:"price"`
	Qty         float64 `json:"qty"`
	Subtotal    float64 `json:"subtotal"`
}

type GetTransactionResponse struct {
	ID               uint                            `json:"id"`
	UserID           uint                            `json:"user_id"`
	CustomerID       *uint                           `json:"customer_id,omitempty"`
	TypePayment      string                          `json:"type_payment"`
	TotalTransaction float64                         `json:"total_transaction"`
	TotalLaba        float64                         `json:"total_laba"`
	CreatedAt        string                          `json:"created_at"`
	DetailsTransaksi []GetProductTransactionResponse `json:"details"`
}
