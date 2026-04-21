package usecase

import (
	"context"
	"shop_project_be/internal/domain"
	"shop_project_be/internal/dto/request"
	"shop_project_be/internal/dto/response"

	"go.uber.org/zap"
)

type customerUsecase struct {
	customerRepo domain.CustomerRepository
	log          *zap.Logger
}

func NewCustomerUsecase(customerRepo domain.CustomerRepository, log *zap.Logger) domain.CustomerUsecase {
	return &customerUsecase{
		customerRepo: customerRepo,
		log:          log,
	}
}

// AddCustomerShop implements [domain.CustomerUsecase].
func (c *customerUsecase) AddCustomerShop(ctx context.Context, request *request.AddCustomerRequest) error {
	panic("unimplemented")
}

// DeleteCustomerShop implements [domain.CustomerUsecase].
func (c *customerUsecase) DeleteCustomerShop(ctx context.Context, request *request.DeleteCustomer) error {
	panic("unimplemented")
}

// GetCustomerShop implements [domain.CustomerUsecase].
func (c *customerUsecase) GetCustomerShop(ctx context.Context, request *request.GetCustomer) (*response.CustomerDtoResponse, error) {
	panic("unimplemented")
}

// GetListCustomerShop implements [domain.CustomerUsecase].
func (c *customerUsecase) GetListCustomerShop(ctx context.Context, request *request.GetAllCustomer) (*[]response.CustomerDtoResponse, error) {
	panic("unimplemented")
}

// UpdateCustomerShop implements [domain.CustomerUsecase].
func (c *customerUsecase) UpdateCustomerShop(ctx context.Context, request *request.UpdateCustomer) error {
	panic("unimplemented")
}
