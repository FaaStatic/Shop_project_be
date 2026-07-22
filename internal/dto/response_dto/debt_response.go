package responsedto

type DebtResponseDto struct {
	NameCustomer    string                `json:"name_customer"`
	UserName        string                `json:"user_name"`
	TotalDebt       string                `json:"total_debt"`
	RemainingDebt   string                `json:"remaining_debt"`
	DateDebt        *string               `json:"date_debt"`
	TransactionList []TransactionResponse `json:"transaction_list"`
}

type DebtListReponseDto struct {
	AfterId         string            `json:"after_id"`
	AfterTime       string            `json:"after_time"`
	HasNext         bool              `json:"has_next"`
	Page            int               `json:"page"`
	TransactionList []DebtResponseDto `json:"transaction_list"`
}

type PrintDebtCustomerResponse struct {
	CustomerName string `json:"customer_name"`
	DebtId       string `json:"debt_id"`
	UrlPdf       string `json:"url_pdf"`
}

// DebtPaymentResponse is the cash debt payment receipt (struk): everything
// the customer needs as proof of payment — how much was owed before, how
// much was paid just now, and how much remains. Rendering it as a PDF/printed
// receipt is the frontend's job; this only supplies the raw numbers.
type DebtPaymentResponse struct {
	DebtId                string `json:"debt_id"`
	CustomerName          string `json:"customer_name"`
	NominalBayar          string `json:"nominal_bayar"`
	PreviousRemainingDebt string `json:"previous_remaining_debt"`
	RemainingDebt         string `json:"remaining_debt"`
	TotalDebt             string `json:"total_debt"`
	Status                string `json:"status"`
	PaidAt                string `json:"paid_at"`
}
