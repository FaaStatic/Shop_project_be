package repository

import (
	"context"
	"errors"
	"fmt"
	"shop_project_be/internal/domain"

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
	result := c.db.WithContext(ctx).Where("customer_id = ?", customerId).Pluck("id", &debtId)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get debt: %w", result.Error)
	}
	return &debtId, nil
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
// Mengambil daftar customer dengan pencarian nama (opsional) dan pagination
// offset (limit + offset). Diurut dari yang terbaru.
func (c *customerRepository) GetAllCustomer(ctx context.Context, search string, limit int, offset int) ([]*domain.Customers, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	query := c.db.WithContext(ctx).Model(&domain.Customers{})
	if search != "" {
		query = query.Where("name LIKE ?", "%"+search+"%")
	}

	var items []*domain.Customers
	result := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&items)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get customers: %w", result.Error)
	}
	return items, nil
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
