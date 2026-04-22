package usecase

import (
	"context"
	"shop_project_be/internal/domain"
	requestdto "shop_project_be/internal/dto/request_dto"
	responsedto "shop_project_be/internal/dto/response_dto"

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
func (c *customerUsecase) AddCustomerShop(ctx context.Context, request *requestdto.AddCustomer) error {
	panic("unimplemented")
}

// DeleteCustomerShop implements [domain.CustomerUsecase].
func (c *customerUsecase) DeleteCustomerShop(ctx context.Context, request *requestdto.DeleteCustomer) error {
	panic("unimplemented")
}

// GetCustomerShop implements [domain.CustomerUsecase].
func (c *customerUsecase) GetCustomerShop(ctx context.Context, request *requestdto.GetCustomer) (*responsedto.CustomerDtoResponse, error) {
	panic("unimplemented")
}

// GetListCustomerShop implements [domain.CustomerUsecase].
func (c *customerUsecase) GetListCustomerShop(ctx context.Context, request *requestdto.GetAllCustomer) (*[]responsedto.CustomerDtoResponse, error) {
	panic("unimplemented")
}

// UpdateCustomerShop implements [domain.CustomerUsecase].
func (c *customerUsecase) UpdateCustomerShop(ctx context.Context, request *requestdto.UpdateCustomer) error {
	panic("unimplemented")
}
