package repository

import (
	"context"
	"errors"
	"fmt"
	"shop_project_be/internal/constant/paginated"
	"shop_project_be/internal/domain"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type customerRepository struct {
	db *gorm.DB
}

func NewCustomerRepository(db *gorm.DB) domain.CustomerRepository {
	return &customerRepository{db: db}
}

// GetDebtIdByCustomerId implements [domain.CustomerRepository].
func (c *customerRepository) GetDebtIdByCustomerId(ctx context.Context, customerId uuid.UUID) (*uuid.UUID, error) {
	var debtId uuid.UUID
	// REQUIRED .Model(&domain.Debts{}): Pluck without a model errors with "table not set".
	result := c.db.WithContext(ctx).Model(&domain.Debts{}).Where("customer_id = ?", customerId).Pluck("id", &debtId)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get debt: %w", result.Error)
	}
	return &debtId, nil
}

// ExistsCustomer implements [domain.CustomerRepository]. Counts by id only so
// no row data or associations are loaded (soft-deleted rows are excluded by the
// default gorm scope).
func (c *customerRepository) ExistsCustomer(ctx context.Context, id uuid.UUID) (bool, error) {
	var count int64
	if err := c.db.WithContext(ctx).Model(&domain.Customers{}).
		Where("id = ?", id).Limit(1).Count(&count).Error; err != nil {
		return false, internalErr(fmt.Errorf("failed to check customer: %w", err))
	}
	return count > 0, nil
}

// AddCustomer implements [domain.CustomerRepository].
func (c *customerRepository) AddCustomer(ctx context.Context, customer *domain.Customers) error {
	result := c.db.WithContext(ctx).Create(customer)
	if result.Error != nil {
		return fmt.Errorf("failed to add customer: %w", result.Error)
	}
	return nil
}

// DeleteCustomer implements [domain.CustomerRepository].
func (c *customerRepository) DeleteCustomer(ctx context.Context, id uuid.UUID) error {
	result := c.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Customers{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete customer: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("customer with id %s not found", id)
	}
	return nil
}

// GetCustomer implements [domain.CustomerRepository].
func (c *customerRepository) GetCustomer(ctx context.Context, id uuid.UUID) (*[]domain.Customers, error) {
	var customers []domain.Customers
	result := c.db.Preload("Transactions").Preload("Debts").WithContext(ctx).Where("id = ?", id).First(&customers)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("customer with id %s not found: %w", id, result.Error)
		}
		return nil, fmt.Errorf("failed to get customer: %w", result.Error)
	}
	if len(customers) == 0 {
		return nil, fmt.Errorf("customer with id %s not found", id)
	}
	return &customers, nil
}

// GetAllCustomer implements [domain.CustomerRepository].
// Fetches the customer list with optional name search and cursor
// pagination (created_at + id as tie-breaker). Fetch limit+1 rows to
// detect whether there is a next page (has_next).
func (c *customerRepository) GetAllCustomer(ctx context.Context, filter domain.FilterCustomer) (*domain.CustomersPaginated, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 10
	}

	order := "DESC"
	if strings.ToUpper(filter.Order) == "ASC" {
		order = "ASC"
	}

	query := c.db.WithContext(ctx).Model(&domain.Customers{})
	if filter.Search != "" {
		escaped := strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(filter.Search)
		query = query.Where("name LIKE ? ESCAPE '\\'", "%"+escaped+"%")
	}

	if filter.Cursor != nil {
		if order == "ASC" {
			query = query.Where("(created_at > ?) OR (created_at = ? AND id > ?)",
				filter.Cursor.AfterTime,
				filter.Cursor.AfterTime,
				filter.Cursor.AfterID,
			)
		} else {
			query = query.Where("(created_at < ?) OR (created_at = ? AND id < ?)",
				filter.Cursor.AfterTime,
				filter.Cursor.AfterTime,
				filter.Cursor.AfterID,
			)
		}
	}

	var items []*domain.Customers
	result := query.Order("created_at " + order + ", id " + order).Limit(filter.Limit + 1).Find(&items)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get customers: %w", result.Error)
	}

	hasNext := len(items) > filter.Limit
	if hasNext {
		items = items[:filter.Limit]
	}
	var nextCursor *paginated.CursorMeta
	if hasNext && len(items) > 0 {
		last := items[len(items)-1]
		nextCursor = &paginated.CursorMeta{
			AfterTime: last.CreatedAt,
			AfterID:   last.ID,
		}
	}

	return &domain.CustomersPaginated{
		DataItem: items,
		HasNext:  hasNext,
		Cursor:   nextCursor,
	}, nil
}

// UpdateCustomer implements [domain.CustomerRepository].
func (c *customerRepository) UpdateCustomer(ctx context.Context, id uuid.UUID, customer *domain.Customers) error {
	result := c.db.WithContext(ctx).Model(&domain.Customers{}).Where("id = ?", id).Updates(customer)
	if result.Error != nil {
		return fmt.Errorf("failed to update customer: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("customer with id %s not found", id)
	}
	return nil
}
