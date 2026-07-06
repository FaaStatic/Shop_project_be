package requestdto

type DebtPayment struct {
	DebtID       string  `json:"debt_id" validate:"required"`
	UserID       string  `json:"user_id" validate:"required"`
	NominalBayar float64 `json:"nominal_bayar" validate:"required,gt=0"`
}

type GetDebtRequest struct {
	DebtId string `query:"debt_id" validate:"required"`
}

type FilterDebtRequest struct {
	UserId     string  `query:"user_id" validate:"required"`
	CustomerId string  `query:"customer_id"`
	Month      string  `query:"month"`
	Year       string  `query:"year"`
	Limit      int     `query:"limit"`
	Order      string  `query:"order"`
	AfterID    *string `query:"after_id,omitempty"`
	AfterTime  *string `query:"after_time,omitempty"`
}

type AddDebtRequest struct {
	UserId         string  `json:"user_id" validate:"required"`
	CustomerID     string  `json:"customer_id" validate:"required"`
	TotalTransaksi float64 `json:"total_transaksi" validate:"required,gt=0"`
	JatuhTempo     string  `json:"jatuh_tempo" validate:"required"`
}

type DeleteDebtRequest struct {
	UserId string `json:"user_id" validate:"required"`
	DebtId string `json:"debt_id" validate:"required"`
}

type PrintDebtReport struct {
	UserId       string `json:"user_id" validate:"required"`
	DebtId       string `json:"debt_id,omitempty"`
	NameCustomer string `json:"name_customer,omitempty"`
	Month        string `json:"month" validate:"required"`
	Year         string `json:"year" validate:"required"`
}
