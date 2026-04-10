package response

type AddTransactionResponse struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
}

type ProductTransactionResponse struct {
	ProductID   uint    `json:"product_id"`
	ProductName string  `json:"product_name"`
	Price       float64 `json:"price"`
	Qty         float64 `json:"qty"`
	Subtotal    float64 `json:"subtotal"`
}

type TransactionResponse struct {
	TransactionID      uint                         `json:"trx_id"`
	InvoiceNumber      string                       `json:"invoice_number"`
	PaymentType        string                       `json:"payment_type"`
	TotalTransaction   float64                      `json:"total_transaction"`
	TotalLaba          float64                      `json:"total_laba"`
	CreatedAt          string                       `json:"created_at"`
	TransactionDetails []ProductTransactionResponse `json:"transaction_details"`
}

type GetAllTransactionResponse struct {
	UserID          uint                         `json:"user_id"`
	TransactionList []ProductTransactionResponse `json:"list_transaction"`
}

type PrintReportTransactionResponse struct {
	ID        uint   `json:"id"`
	NoInvoice string `json:"number_invoice"`
	UrlPdf    string `json:"url_pdf"`
}

type PrintReportMonthTransactionResponse struct {
	ID     uint   `json:"id"`
	Month  string `json:"month"`
	Year   string `json:"Year"`
	UrlPdf string `json:"url_pdf"`
}
