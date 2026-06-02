package responsedto

import "github.com/google/uuid"

type AddTransactionResponse struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
}

type ProductTransactionResponse struct {
	ProductID   uuid.UUID `json:"product_id"`
	ProductName string    `json:"product_name"`
	Price       float64   `json:"price"`
	Qty         float64   `json:"qty"`
	Subtotal    float64   `json:"subtotal"`
}

type TransactionResponse struct {
	TransactionID      uuid.UUID                     `json:"trx_id"`
	InvoiceNumber      string                        `json:"invoice_number"`
	PaymentType        int                           `json:"payment_type"`
	TotalTransaction   float64                       `json:"total_transaction"`
	TotalProfit        float64                       `json:"total_profit"`
	CreatedAt          string                        `json:"created_at"`
	TransactionDetails []*ProductTransactionResponse `json:"transaction_details"`
}

type GetAllTransactionResponse struct {
	UserID          uuid.UUID                     `json:"user_id"`
	TransactionList []*ProductTransactionResponse `json:"list_transaction"`
}

type PrintReportTransactionResponse struct {
	ID        uuid.UUID `json:"id"`
	NoInvoice string    `json:"number_invoice"`
	UrlPdf    string    `json:"url_pdf"`
}

type PrintReportMonthTransactionResponse struct {
	ID     uuid.UUID `json:"id"`
	Month  string    `json:"month"`
	Year   string    `json:"Year"`
	UrlPdf string    `json:"url_pdf"`
}
