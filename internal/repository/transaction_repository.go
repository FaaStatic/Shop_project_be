package repository

import (
	"context"
	"shop_project_be/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type transactionRepository struct {
	db *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) domain.TransactionRepository {
	return &transactionRepository{db: db}
}

// CreateTransaction implements [domain.TransactionRepository].
func (t *transactionRepository) CreateTransaction(ctx context.Context, transaction *domain.Transactions) error {
	panic("unimplemented")
}

// DeleteTransaction implements [domain.TransactionRepository].
func (t *transactionRepository) DeleteTransaction(ctx context.Context, id uuid.UUID) error {
	panic("unimplemented")
}

// GetAllTransaction implements [domain.TransactionRepository].
func (t *transactionRepository) GetAllTransaction(ctx context.Context) ([]domain.Transactions, error) {
	panic("unimplemented")
}

// GetTransactionByID implements [domain.TransactionRepository].
func (t *transactionRepository) GetTransactionByID(ctx context.Context, id uuid.UUID) (*domain.Transactions, error) {
	panic("unimplemented")
}

// UpdateTransaction implements [domain.TransactionRepository].
func (t *transactionRepository) UpdateTransaction(ctx context.Context, id uuid.UUID, trx *domain.Transactions) error {
	panic("unimplemented")
}
