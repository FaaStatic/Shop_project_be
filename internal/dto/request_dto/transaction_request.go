package requestdto

type AddTransactionRequest struct {
	NoInvoice   string `json:"no_invoice" validate:"required"`
	TypePayment string `json:"type_payment" validate:"required,oneof=tunai hutang transfer qris"`
	// TotalTransaction is informational only; the final value is computed server-side from
	// the product price (see usecase.AddTransaction) so it cannot be manipulated.
	TotalTransaction float64                       `json:"total_price,omitempty"`
	UserId           string                        `json:"user_id" validate:"required,uuid"`
	CustomerId       *string                       `json:"customer_id,omitempty" validate:"omitempty,uuid"`
	Details          []AddTransactionDetailRequest `json:"details" validate:"required,min=1,dive"`
}

type AddTransactionDetailRequest struct {
	ProductId string  `json:"product_id" validate:"required,uuid"`
	Qty       float64 `json:"qty" validate:"required,gt=0"`
	// Subtotal is ignored by the server (computed from product price × qty); left
	// optional for backward compatibility with older payloads.
	Subtotal float64 `json:"subtotal,omitempty"`
}

type GetTransactionRequest struct {
	ID         string `query:"id" validate:"required"`
	UserId     string `query:"user_id" validate:"required"`
	CustomerId string `query:"customer_id,omitempty"`
}

type FilterTransactionRequest struct {
	UserId        string  `query:"user_id" validate:"required"`
	DateStart     *string `query:"date_start,omitempty"`
	DateEnd       *string `query:"date_end,omitempty"`
	TypePayment   int     `query:"type_payment" validate:"required,oneof=0 1 2 3"`
	InvoiceNumber string  `query:"number_invoices,omitempty"`
	AfterTime     *string `query:"after_time,omitempty"`
	AfterID       *string `query:"after_id,omitempty"`
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
