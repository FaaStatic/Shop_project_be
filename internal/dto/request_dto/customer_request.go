package requestdto

type AddCustomer struct {
	UserId       string `json:"user_id" validate:"required,uuid"`
	CustomerName string `json:"customer_name" validate:"required"`
	PhoneNumber  string `json:"phone_number,omitempty"`
	Address      string `json:"address,omitempty"`
}
type UpdateCustomer struct {
	CustomerId   string `json:"customer_id" validate:"required,uuid"`
	UserId       string `json:"user_id" validate:"required,uuid"`
	CustomerName string `json:"customer_name,omitempty"`
	PhoneNumber  string `json:"phone_number,omitempty"`
	Address      string `json:"address,omitempty"`
}

type DeleteCustomer struct {
	CustomerId string `json:"customer_id" validate:"required,uuid"`
	UserId     string `json:"user_id" validate:"required,uuid"`
}

type GetAllCustomer struct {
	UserId    string  `json:"user_id" validate:"required,uuid"`
	Limit     int     `query:"limit"`
	Search    string  `query:"search"`
	Order     string  `query:"order"`
	AfterID   *string `query:"after_id,omitempty"`
	AfterTime *string `query:"after_time,omitempty"`
}

type GetCustomer struct {
	CustomerId string `query:"customer_id" validate:"required,uuid"`
}
