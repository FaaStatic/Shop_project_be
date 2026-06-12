package repository

import (
	"context"
	"errors"
	"fmt"
	"shop_project_be/internal/domain"
	"shop_project_be/pkg/dbtx"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type customerRepository struct {
	db *gorm.DB
}

func NewCustomerRepository(db *gorm.DB) domain.CustomerRepository {
	return &customerRepository{db: db}
}

// GetDebtIdByCustomerId implements [domain.CustomerRepository].
//
// Mengembalikan id hutang milik customer, atau (nil, nil) bila customer belum
// punya hutang. Sebelumnya memakai Pluck ke satu uuid sehingga saat tidak ada
// baris ia tetap mengembalikan pointer ke uuid.Nil (tidak pernah nil) — itu
// membuat alur "hutang baru vs nambah" salah. Sekarang pakai First + cek
// ErrRecordNotFound. dbtx.Conn membuatnya ikut transaksi penjualan bila ada.
func (c *customerRepository) GetDebtIdByCustomerId(ctx context.Context, customerId uuid.UUID) (*uuid.UUID, error) {
	var debt domain.Debts
	result := dbtx.Conn(ctx, c.db).WithContext(ctx).
		Select("id").Where("customer_id = ?", customerId).First(&debt)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get debt: %w", result.Error)
	}
	return &debt.ID, nil
}

// LockCustomerForUpdate implements [domain.CustomerRepository].
//
// Mengunci baris customer (SELECT ... FOR UPDATE) di dalam transaksi berjalan.
// Dipakai saat penjualan hutang: dengan mengunci customer-nya lebih dulu, dua
// transaksi hutang bersamaan untuk customer yang sama akan ANTRE — sehingga
// pengecekan "hutang baru vs nambah" tidak balapan dan tidak membuat baris
// hutang ganda.
func (c *customerRepository) LockCustomerForUpdate(ctx context.Context, id uuid.UUID) error {
	var customer domain.Customers
	result := dbtx.Conn(ctx, c.db).WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("id").Where("id = ?", id).First(&customer)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return fmt.Errorf("customer with id %s not found", id)
		}
		return fmt.Errorf("failed to lock customer: %w", result.Error)
	}
	return nil
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
