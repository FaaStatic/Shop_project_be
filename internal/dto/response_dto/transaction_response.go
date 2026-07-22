package responsedto

import "github.com/google/uuid"

// AddTransactionResponse is returned after a sale is recorded. DebtInfo is
// present only when the sale is hutang (debt) — a cash/transfer/qris sale
// leaves it nil since no debt was touched.
type AddTransactionResponse struct {
	TransactionID    string               `json:"transaction_id"`
	NoInvoice        string               `json:"no_invoice"`
	TotalTransaction float64              `json:"total_transaction"`
	PaymentType      string               `json:"payment_type"`
	DebtInfo         *DebtTransactionInfo `json:"debt_info,omitempty"`
}

// DebtTransactionInfo is the debt side-effect of a single hutang sale: how
// much the customer owed before this sale, how much this sale added, and
// what they owe now — the numbers a receipt needs.
type DebtTransactionInfo struct {
	DebtID                string `json:"debt_id"`
	PreviousRemainingDebt string `json:"previous_remaining_debt"`
	AmountAdded           string `json:"amount_added"`
	TotalDebt             string `json:"total_debt"`
	RemainingDebt         string `json:"remaining_debt"`
	Status                string `json:"status"`
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
	UserID          string                 `json:"user_id"`
	AfterId         string                 `json:"after_id"`
	AfterTime       string                 `json:"after_time"`
	HasNext         bool                   `json:"has_next"`
	TransactionList []*TransactionResponse `json:"list_transaction"`
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
