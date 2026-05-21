package requestdto

type AddTransactionRequest struct {
	NoInvoice        string                        `json:"no_invoice" validate:"required"`
	TypePayment      string                        `json:"type_payment" validate:"required,oneof=tunai hutang transfer qris"`
	TotalTransaction float64                       `json:"total_price" validate:"required,gt=0"`
	UserId           string                        `json:"user_id" validate:"required"`
	CustomerId       *string                       `json:"customer_id,omitempty"`
	Details          []AddTransactionDetailRequest `json:"details" validate:"required,dive"`
}

type AddTransactionDetailRequest struct {
	ProductId string  `json:"product_id" validate:"required"`
	Qty       float64 `json:"qty" validate:"required,gt=0"`
	Subtotal  float64 `json:"subtotal" validate:"required,gt=0"`
}

type GetTransactionRequest struct {
	ID         uint `query:"id" validate:"required"`
	UserId     uint `query:"user_id" validate:"required"`
	CustomerId uint `query:"customer_id,omitempty"`
}

type FilterTransactionRequest struct {
	UserId        string `query:"user_id" validate:"required"`
	DateStart     string `query:"date_start,omitempty"`
	DateEnd       string `query:"date_end,omitempty"`
	TypePayment   int    `query:"type_payment" validate:"required,oneof=0 1 2 3"`
	SearchInvoice string `query:"number_invoices",omitempty"`
}

type DeleteTransactionRequest struct {
	ID string `json:"trx_id" validate:"required"`
}

type PrintReportTransactionRequest struct {
	UserId    string `query:"user_id" validate:"required"`
	TrxId     string `query:"trx_id,omitempty"`
	NoInvoice string `query:"number_invoice,omitempty"`
}

type PrintReportMonthRequest struct {
	UserId string `query:"user_id" validate:"required"`
	Month  int    `query:"month"`
	Year   int    `query:"year"`
}
