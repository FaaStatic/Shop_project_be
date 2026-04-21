package repository

import (
	"context"
	"shop_project_be/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type debtRepository struct {
	db *gorm.DB
}

func NewDebtRepository(db *gorm.DB) domain.DebtRepository {
	return &debtRepository{db: db}
}

// AddDebt implements [domain.DebtRepository].
func (d *debtRepository) AddDebt(ctx context.Context, debt *domain.Debts) error {
	panic("unimplemented")
}

// DeleteDebt implements [domain.DebtRepository].
func (d *debtRepository) DeleteDebt(ctx context.Context, id uuid.UUID) error {
	panic("unimplemented")
}

// GetAllDebt implements [domain.DebtRepository].
func (d *debtRepository) GetAllDebt(ctx context.Context) (*[]domain.Debts, error) {
	panic("unimplemented")
}
