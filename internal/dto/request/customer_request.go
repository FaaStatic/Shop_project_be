package request

type AddCustomerRequest struct {
	UserId       uint   `json:"user_id" validate="required"`
	CustomerName string `json:"customer_name" validate="required"`
	PhoneNumber  string `json:"phone_number,omitempty"`
	Address      string `json:"address,omitempty"`
}
type UpdateCustomer struct {
	CustomerId   uint   `json:"customer_id" validate="required"`
	UserId       uint   `json:"user_id" validate="required"`
	CustomerName string `json:"customer_name,omitempty"`
	PhoneNumber  string `json:"phone_number,omitempty"`
	Address      string `json:"address,omitempty"`
}

type DeleteCustomer struct {
	CustomerId uint `json:"customer_id" validate="required"`
	UserId     uint `json:"user_id" validate="required"`
}

type GetAllCustomer struct {
	UserId uint   `json:"user_id" validate="required"`
	Page   int    `query:"page"`
	Limit  int    `query:"limit"`
	Search string `query:"search"`
}

type GetCustomer struct {
	CustomerId uint `json:"customer_id" validate="required"`
}
