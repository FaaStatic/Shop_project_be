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
	"gorm.io/gorm/clause"
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
	result := d.db.Preload("Customer").Preload("Transactions").Preload("DebtPayments").Preload("DebtPayments.User").WithContext(ctx).Where("id = ?", id).First(&debt)
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

// PayDebt implements [domain.DebtRepository].
// Preloads Customer (a receipt needs the customer's name) alongside the
// locked debt row.
func (d *debtRepository) PayDebt(ctx context.Context, debtID uuid.UUID, payment *domain.DebtPayments) (*domain.DebtPaymentResult, error) {
	var result domain.DebtPaymentResult
	err := runTxDB(ctx, d.db, func(tx *gorm.DB) error {
		var debt domain.Debts
		if err := tx.Preload("Customer").Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", debtID).First(&debt).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("debt with id %s not found", debtID)
			}
			return internalErr(fmt.Errorf("failed to lock debt: %w", err))
		}
		if debt.RemainingDebt <= 0 || debt.Status == enum.LUNAS {
			return fmt.Errorf("debt has already been fully paid")
		}
		if payment.NominalBayar > debt.RemainingDebt {
			return fmt.Errorf("payment amount (%.2f) exceeds remaining debt (%.2f)", payment.NominalBayar, debt.RemainingDebt)
		}
		previousRemaining := debt.RemainingDebt

		newRemaining := debt.RemainingDebt - payment.NominalBayar
		status := enum.BELUM_LUNAS
		if newRemaining <= 0 {
			newRemaining = 0
			status = enum.LUNAS
		}
		if err := tx.Model(&domain.Debts{}).Where("id = ?", debt.ID).
			Updates(map[string]interface{}{
				"remaining_debt": newRemaining,
				"status":         status,
			}).Error; err != nil {
			return internalErr(fmt.Errorf("failed to update debt: %w", err))
		}

		payment.DebtID = debt.ID
		if err := tx.Create(payment).Error; err != nil {
			return internalErr(fmt.Errorf("failed to record debt payment: %w", err))
		}

		debt.RemainingDebt = newRemaining
		debt.Status = status
		result = domain.DebtPaymentResult{
			Debt:                  &debt,
			PreviousRemainingDebt: previousRemaining,
			PaymentID:             payment.ID,
			PaidAt:                payment.TanggalBayar,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetAllDebt implements [domain.DebtRepository].
func (d *debtRepository) GetAllDebt(ctx context.Context, filter domain.FilterDebt) (*domain.DebtsPaginated, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 10
	}

	order := "DESC"
	if strings.ToUpper(filter.Order) == "ASC" {
		order = "ASC"
	}

	query := d.db.Preload("Customer").Preload("Transactions").Preload("DebtPayments").WithContext(ctx).Model(&domain.Debts{})

	// Search matches the customer name. The JOIN needs an explicit Select("debts.*")
	// so columns sharing a name in both tables (id, created_at, etc.) are not
	// ambiguous/overwritten when scanned into the Debts struct.
	if filter.Search != "" {
		escaped := strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(filter.Search)
		query = query.Select("debts.*").
			Joins("JOIN customers ON customers.id = debts.customer_id AND customers.deleted_at IS NULL").
			Where("customers.name LIKE ? ESCAPE '\\'", "%"+escaped+"%")
	}

	if filter.Cursor != nil {
		if order == "ASC" {
			query = query.Where("(debts.created_at > ?) OR (debts.created_at = ? AND debts.id > ?)",
				filter.Cursor.AfterTime,
				filter.Cursor.AfterTime,
				filter.Cursor.AfterID,
			)
		} else {
			query = query.Where("(debts.created_at < ?) OR (debts.created_at = ? AND debts.id < ?)",
				filter.Cursor.AfterTime,
				filter.Cursor.AfterTime,
				filter.Cursor.AfterID,
			)
		}
	}

	if filter.CustomerID != uuid.Nil {
		query = query.Where("debts.customer_id = ?", filter.CustomerID)
	}
	if filter.Status != nil {
		query = query.Where("debts.status = ?", *filter.Status)
	}

	var itemList []*domain.Debts

	result := query.Order("debts.created_at " + order + ", debts.id " + order).Limit(filter.Limit + 1).Find(&itemList)
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
