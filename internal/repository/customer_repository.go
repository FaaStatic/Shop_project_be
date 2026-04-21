package repository

import (
	"context"
	"shop_project_be/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type customerRepository struct {
	db *gorm.DB
}

// AddCustomer implements [domain.CustomerRepository].
func (c *customerRepository) AddCustomer(ctx context.Context, customer *domain.Customers) error {
	panic("unimplemented")
}

// DeleteCustomer implements [domain.CustomerRepository].
func (c *customerRepository) DeleteCustomer(ctx context.Context, id uuid.UUID) error {
	panic("unimplemented")
}

// GetCustomer implements [domain.CustomerRepository].
func (c *customerRepository) GetCustomer(ctx context.Context) (*[]domain.Customers, error) {
	panic("unimplemented")
}

// UpdateCustomer implements [domain.CustomerRepository].
func (c *customerRepository) UpdateCustomer(ctx context.Context, id uuid.UUID, customer *domain.Customers) error {
	panic("unimplemented")
}

func NewCustomerRepository(db *gorm.DB) domain.CustomerRepository {
	return &customerRepository{db: db}
}
