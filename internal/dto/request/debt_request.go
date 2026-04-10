package request

type DebtPayment struct {
	DebtID       uint    `json:"debt_id" validate:"required"`
	UserID       uint    `json:"user_id" validate:"required"`
	NominalBayar float64 `json:"nominal_bayar" validate:"required,gt=0"`
}

type GetDebtRequest struct {
	DebtId uint `query:"debt_id" validate="required"`
}

type GetAllDebtRequest struct {
	UserId     uint `query:"user_id" validate="required"`
	CustomerId uint `query:"customer_id" validate="required"`
}

type FilterDebtRequest struct {
	UserId     uint   `query:"user_id" validate="required"`
	CustomerId uint   `query:"customer_id"`
	Month      string `query:"month"`
	Year       string `query:"year"`
}

type AddDebtRequest struct {
	UserId         uint    `json:"user_id" validate:"required"`
	CustomerID     uint    `json:"customer_id" validate:"required"`
	TotalTransaksi float64 `json:"total_transaksi" validate:"required,gt=0"`
	JatuhTempo     string  `json:"jatuh_tempo" validate:"required"`
}

type DeleteDebtRequest struct {
	UserId uint `json:"user_id" validate:"required"`
	DebtId uint `json:"debt_id" validate="required"`
}

type PrintDebtReport struct {
	UserId string `json:"user_id" validate:"required"`
	Month  string `json:"month" validate:"required"`
	Year   string `json:"year" validate:"required"`
}
