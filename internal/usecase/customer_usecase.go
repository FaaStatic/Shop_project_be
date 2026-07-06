package usecase

import (
	"context"
	"fmt"
	"shop_project_be/internal/constant/paginated"
	"shop_project_be/internal/domain"
	requestdto "shop_project_be/internal/dto/request_dto"
	responsedto "shop_project_be/internal/dto/response_dto"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// toCustomerResponse maps a Customers entity to the response DTO.
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
// Fetches the customer list with name search and cursor pagination
// (after_id + after_time from the previous page's result).
func (c *customerUsecase) GetListCustomerShop(ctx context.Context, request *requestdto.GetAllCustomer) (*responsedto.ListCustomerDtoResponse, error) {
	// Cursor is optional. The first page has no after_id/after_time yet, so
	// both must be set for the cursor to apply; otherwise leave it nil so the
	// repo does not filter created_at with a zero-time (which empties the result).
	var afterId, afterTimeRaw string
	if request.AfterID != nil {
		afterId = strings.TrimSpace(*request.AfterID)
	}
	if request.AfterTime != nil {
		afterTimeRaw = strings.TrimSpace(*request.AfterTime)
	}

	var cursor *paginated.CursorMeta
	if afterId != "" && afterTimeRaw != "" {
		afterTime, err := time.Parse(paginated.TimeLayout, afterTimeRaw)
		if err != nil {
			c.log.Error("failed to parse after_time", zap.Error(err))
			return nil, fmt.Errorf("invalid after_time format")
		}
		afterUUID, err := uuid.Parse(afterId)
		if err != nil {
			c.log.Error("failed to parse after_id", zap.Error(err))
			return nil, fmt.Errorf("invalid after_id format")
		}
		cursor = &paginated.CursorMeta{AfterTime: afterTime, AfterID: afterUUID}
	}

	filter := domain.FilterCustomer{
		Search: request.Search,
		Cursor: cursor,
		Limit:  request.Limit,
		Order:  request.Order,
	}

	result, err := c.customerRepo.GetAllCustomer(ctx, filter)
	if err != nil {
		c.log.Error("failed to get customers", zap.Error(err))
		return nil, fmt.Errorf("failed to get customers")
	}

	responses := make([]responsedto.CustomerDtoResponse, 0, len(result.DataItem))
	for _, item := range result.DataItem {
		responses = append(responses, toCustomerResponse(item))
	}

	// Cursor is nil on the last page; Encode is nil-safe (no panic).
	nextId, nextTime := result.Cursor.Encode()

	return &responsedto.ListCustomerDtoResponse{
		AfterId:      nextId,
		AfterTime:    nextTime,
		HasNext:      result.HasNext,
		CustomerList: responses,
	}, nil
}

// UpdateCustomerShop implements [domain.CustomerUsecase].
func (c *customerUsecase) UpdateCustomerShop(ctx context.Context, request *requestdto.UpdateCustomer) error {
	id, err := uuid.Parse(request.CustomerId)
	if err != nil {
		c.log.Error("failed to parse customer id", zap.Error(err))
		return fmt.Errorf("invalid customer id format")
	}

	// Only populated fields are updated (Updates with a struct ignores zero
	// values), so partial updates are safe.
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
