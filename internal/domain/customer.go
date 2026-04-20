package domain

import (
	"context"
	"shop_project_be/internal/dto/request"
	"shop_project_be/internal/dto/response"

	"github.com/google/uuid"
)

type Customers struct {
	ID      uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name    string    `gorm:"type:varchar(150);not null" json:"name"`
	Phone   string    `gorm:"type:varchar(15)" json:"phone"`
	Address string    `gorm:"type:text" json:"address"`

	Transactions []Transactions `gorm:"foreignKey:CustomerID" json:"transactions,omitempty"`
	Debts        []Debts        `gorm:"foreignKey:CustomerID" json:"debts,omitempty"`
}

func (c *Customers) TableName() string {
	return "customers"
}

type CustomerRepository interface {
	GetCustomer(ctx *context.Context) (*[]Customers, error)
	UpdateCustomer(ctx *context.Context, id uuid.UUID, customer *Customers) error
	AddCustomer(ctx *context.Context, customer *Customers) error
	DeleteCustomer(ctx *context.Context, id uuid.UUID) error
}

type CustomerUsecase interface {
	AddCustomerShop(ctx *context.Context, request *request.AddCustomerRequest) error
	UpdateCustomerShop(ctx *context.Context, request *request.UpdateCustomer) error
	GetListCustomerShop(ctx *context.Context, request *request.GetAllCustomer) (*[]response.CustomerDtoResponse, error)
	GetCustomerShop(ctx *context.Context, request *request.GetCustomer) (*response.CustomerDtoResponse, error)
	DeleteCustomerShop(ctx *context.Context, request *request.DeleteCustomer) error
}
