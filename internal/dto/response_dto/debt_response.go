package responsedto

type DebtResponseDto struct {
	NameCustomer    string                `json="name_customer"`
	UserName        string                `json="user_name"`
	TotalDebt       string                `json="total_debt"`
	RemainingDebt   string                `json="remaining_debt"`
	DateDebt        *string               `json="date_debt"`
	TransactionList []TransactionResponse `json="transaction_list"`
}

type PrintDebtCustomerResponse struct {
	CustomerName string `json="customer_name"`
	DebtId       string `json="debt_id"`
	UrlPdf       string `json="url_pdf"`
}
