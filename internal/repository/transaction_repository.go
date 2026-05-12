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

type transactionRepository struct {
	db *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) domain.TransactionRepository {
	return &transactionRepository{db: db}
}

// CreateTransaction implements [domain.TransactionRepository].
func (t *transactionRepository) CreateTransaction(ctx context.Context, transaction *domain.Transactions) error {
	result := t.db.WithContext(ctx).Create(transaction)
	if result.Error != nil {
		return fmt.Errorf("failed to add transaction: %w", result.Error)
	}
	return nil
}

// DeleteTransaction implements [domain.TransactionRepository].
func (t *transactionRepository) DeleteTransaction(ctx context.Context, id uuid.UUID) error {
	result := t.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Transactions{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete product: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("product with id %s not found", id)
	}
	return nil
}

// GetAllTransaction implements [domain.TransactionRepository].
func (t *transactionRepository) GetAllTransaction(ctx context.Context, filter domain.FilterTransaction) (*domain.ResultTransaction, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 10
	}

	order := "ASC"
	if strings.ToUpper(filter.Order) == "DESC" {
		order = "DESC"
	}

	query := t.db.Preload("User").
		Preload("Customer").
		Preload("TransactionDetail").Preload("TransactionDetail.Product").WithContext(ctx).Model(&domain.Transactions{})

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

	var itemList []*domain.Transactions

	result := query.Order("created_at " + order + ", id " + order).Limit(filter.Limit + 1).Find(&itemList)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", result.Error)
	}
	hasNext := len(itemList) > filter.Limit
	if hasNext {
		itemList = itemList[:filter.Limit]
	}
	var nextCursor *paginated.CursorMeta
	if hasNext && len(itemList) > 0 {
		last := itemList[len(itemList)-1]
		nextCursor = &paginated.CursorMeta{
			AfterTime: last.CreatedAt,
			AfterID:   last.ID,
		}
	}
	return &domain.ResultTransaction{
		DataItem: itemList,
		HasNext:  hasNext,
		Cursor:   nextCursor,
	}, nil
}

// GetTransactionByID implements [domain.TransactionRepository].
func (t *transactionRepository) GetTransactionByID(ctx context.Context, id uuid.UUID) (*domain.Transactions, error) {
	var item domain.Transactions
	result := t.db.WithContext(ctx).Preload("User").
		Preload("Customer").
		Preload("TransactionDetail").Preload("TransactionDetail.Product").Where("id = ?", id).First(&item)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("product with id %s not found: %w", id, result.Error)
		}
		return nil, fmt.Errorf("failed to get transaction: %w", result.Error)
	}
	return &item, nil
}

// UpdateTransaction implements [domain.TransactionRepository].
func (t *transactionRepository) UpdateTransaction(ctx context.Context, id uuid.UUID, trx *domain.Transactions) error {
	result := t.db.WithContext(ctx).Model(&domain.Transactions{}).Where("id = ?", id).Updates(trx)

	if result.Error != nil {
		return fmt.Errorf("failed to update product: %w", result.Error)
	}
	return nil
}
