package usecase

import (
	"context"
	"fmt"
	"shop_project_be/internal/domain"
	requestdto "shop_project_be/internal/dto/request_dto"
	responsedto "shop_project_be/internal/dto/response_dto"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// toCustomerResponse memetakan entitas Customers ke DTO response.
func toCustomerResponse(c *domain.Customers) responsedto.CustomerDtoResponse {
	return responsedto.CustomerDtoResponse{
		ID:     c.ID,
		Nama:   c.Name,
		NoHP:   c.Phone,
		Alamat: c.Address,
	}
}

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
	customer := &domain.Customers{
		Name:    request.CustomerName,
		Phone:   request.PhoneNumber,
		Address: request.Address,
	}
	if err := c.customerRepo.AddCustomer(ctx, customer); err != nil {
		c.log.Error("failed to add customer", zap.Error(err))
		return fmt.Errorf("failed to add customer")
	}
	return nil
}

// DeleteCustomerShop implements [domain.CustomerUsecase].
func (c *customerUsecase) DeleteCustomerShop(ctx context.Context, request *requestdto.DeleteCustomer) error {
	id, err := uuid.Parse(request.CustomerId)
	if err != nil {
		c.log.Error("failed to parse customer id", zap.Error(err))
		return fmt.Errorf("invalid customer id format")
	}
	if err := c.customerRepo.DeleteCustomer(ctx, id); err != nil {
		c.log.Error("failed to delete customer", zap.Error(err))
		return fmt.Errorf("failed to delete customer")
	}
	return nil
}

// GetCustomerShop implements [domain.CustomerUsecase].
func (c *customerUsecase) GetCustomerShop(ctx context.Context, request *requestdto.GetCustomer) (*responsedto.CustomerDtoResponse, error) {
	id, err := uuid.Parse(request.CustomerId)
	if err != nil {
		c.log.Error("failed to parse customer id", zap.Error(err))
		return nil, fmt.Errorf("invalid customer id format")
	}

	customers, err := c.customerRepo.GetCustomer(ctx, id)
	if err != nil {
		c.log.Error("failed to get customer", zap.Error(err))
		return nil, fmt.Errorf("failed to get customer")
	}
	if customers == nil || len(*customers) == 0 {
		c.log.Error("customer not found", zap.String("customer_id", request.CustomerId))
		return nil, fmt.Errorf("customer not found")
	}

	response := toCustomerResponse(&(*customers)[0])
	return &response, nil
}

// GetListCustomerShop implements [domain.CustomerUsecase].
func (c *customerUsecase) GetListCustomerShop(ctx context.Context, request *requestdto.GetAllCustomer) (*[]responsedto.CustomerDtoResponse, error) {
	limit := request.Limit
	if limit <= 0 {
		limit = 10
	}
	page := request.Page
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit

	customers, err := c.customerRepo.GetAllCustomer(ctx, request.Search, limit, offset)
	if err != nil {
		c.log.Error("failed to get customers", zap.Error(err))
		return nil, fmt.Errorf("failed to get customers")
	}

	responses := make([]responsedto.CustomerDtoResponse, 0, len(customers))
	for _, item := range customers {
		responses = append(responses, toCustomerResponse(item))
	}
	return &responses, nil
}

// UpdateCustomerShop implements [domain.CustomerUsecase].
func (c *customerUsecase) UpdateCustomerShop(ctx context.Context, request *requestdto.UpdateCustomer) error {
	id, err := uuid.Parse(request.CustomerId)
	if err != nil {
		c.log.Error("failed to parse customer id", zap.Error(err))
		return fmt.Errorf("invalid customer id format")
	}

	// Hanya field yang terisi yang diperbarui (Updates dengan struct mengabaikan
	// nilai zero), sehingga aman untuk update parsial.
	customer := &domain.Customers{
		Name:    request.CustomerName,
		Phone:   request.PhoneNumber,
		Address: request.Address,
	}
	if err := c.customerRepo.UpdateCustomer(ctx, id, customer); err != nil {
		c.log.Error("failed to update customer", zap.Error(err))
		return fmt.Errorf("failed to update customer")
	}
	return nil
}
