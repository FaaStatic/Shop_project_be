package repository

import (
	"context"
	"errors"
	"fmt"
	"shop_project_be/internal/constant/enum"
	"shop_project_be/internal/constant/paginated"
	"shop_project_be/internal/domain"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type debtRepository struct {
	db *gorm.DB
}

func NewDebtRepository(db *gorm.DB) domain.DebtRepository {
	return &debtRepository{db: db}
}

// GetDebtByID implements [domain.DebtRepository].
func (d *debtRepository) GetDebtByID(ctx context.Context, id uuid.UUID) (*domain.Debts, error) {
	var debt domain.Debts
	result := d.db.Preload("Customer").Preload("Transactions").Preload("DebtPayments").WithContext(ctx).Where("id = ?", id).First(&debt)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("debt with id %s not found: %w", id, result.Error)
		}
		return nil, fmt.Errorf("failed to get debt: %w", result.Error)
	}
	return &debt, nil
}

// UpdateDebt implements [domain.DebtRepository].
func (d *debtRepository) UpdateDebt(ctx context.Context, id uuid.UUID, debt *domain.Debts) error {
	result := d.db.WithContext(ctx).Model(&domain.Debts{}).Where("id = ?", id).Updates(debt)
	if result.Error != nil {
		return fmt.Errorf("failed to update debt: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("debt with id %s not found", id)
	}
	return nil
}

// AddDebt implements [domain.DebtRepository].
func (d *debtRepository) AddDebt(ctx context.Context, debt *domain.Debts) error {
	result := d.db.WithContext(ctx).Create(debt)
	if result.Error != nil {
		return fmt.Errorf("failed to add debt: %w", result.Error)
	}
	return nil
}

// DeleteDebt implements [domain.DebtRepository].
func (d *debtRepository) DeleteDebt(ctx context.Context, id uuid.UUID) error {
	result := d.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Debts{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete debt: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("debt with id %s not found", id)
	}
	return nil
}

// GetAllDebt implements [domain.DebtRepository].
func (d *debtRepository) GetAllDebt(ctx context.Context, filter domain.FilterDebt) (*domain.DebtsPaginated, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 10
	}

	order := "ASC"
	if strings.ToUpper(filter.Order) == "DESC" {
		order = "DESC"
	}

	query := d.db.Preload("Customer").Preload("Transactions").Preload("DebtPayments").WithContext(ctx).Model(&domain.Debts{})

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

	if filter.CustomerID != uuid.Nil {
		query = query.Where("customer_id = ?", filter.CustomerID)
	}
	if filter.Status != enum.DebtStatus(0) {
		query = query.Where("status = ?", filter.Status)
	}

	var itemList []*domain.Debts

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
	return &domain.DebtsPaginated{
		Data:    itemList,
		HasNext: hasNext,
		Cursor:  nextCursor,
	}, nil
}
