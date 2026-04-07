package request

type AddTransactionRequest struct {
	NoInvoice        string                        `json:"no_invoice" validate:"required"`
	TypePayment      string                        `json:"type_payment" validate:"required,oneof=tunai hutang transfer qris"`
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
	ID         uint `json:"id" validate:"required"`
	UserId     uint `json:"user_id" validate:"required"`
	CustomerId uint `json:"customer_id,omitempty"`
}

type FilterTransactionRequest struct {
	UserId        uint   `json:"user_id" validate:"required"`
	DateStart     string `json:"date_start,omitempty"`
	DateEnd       string `json:"date_end,omitempty"`
	TypePayment   string `json:"type_payment" validate:"required,oneof=tunai hutang transfer qris"`
	SearchInvoice string `json:"number_invoices",omitempty"`
}

type DeleteTransactionRequest struct {
	ID uint `json:"id" validate:"required"`
}

type PrintTransaction struct {
	ID        uint   `json:"id" validate:"required"`
	NoInvoice string `json:"number_invoice",omitempty"`
}
