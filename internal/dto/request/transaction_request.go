package request

type AddTransactionRequest struct {
	NoInvoice        string                        `json:"no_invoice" validate:"required"`
	TypePayment      string                        `json:"type_payment" validate:"required,oneof=cash credit_card debit_card"`
	TotalTransaction float64                       `json:"total_price" validate:"required,gt=0"`
	UserId           uint                          `json:"user_id" validate:"required"`
	CustomerId       *uint                         `json:"customer_id,omitempty"`
	Details          []AddTransactionDetailRequest `json:"details" validate:"required,dive"`
}

type AddTransactionDetailRequest struct {
	ProductId uint    `json:"product_id" validate:"required"`
	Qty       float64 `json:"qty" validate:"required,gt=0"`
	Subtotal  float64 `json:"subtotal" validate:"required,gt=0"`
}

type GetTransactionRequest struct {
	ID uint `json:"id" validate:"required"`
}

type FilterTransactionRequest struct {
	NoInvoice  string `json:"no_invoice,omitempty"`
	UserId     uint   `json:"user_id,omitempty"`
	CustomerId uint   `json:"customer_id,omitempty"`
}
