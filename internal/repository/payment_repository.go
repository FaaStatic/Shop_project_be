package repository

import (
	"context"
	"errors"
	"fmt"
	"shop_project_be/internal/domain"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type paymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) domain.PaymentRepository {
	return &paymentRepository{db: db}
}

// Create stores a new payment (initial status pending).
func (p *paymentRepository) Create(ctx context.Context, payment *domain.Payment) error {
	if err := p.db.WithContext(ctx).Create(payment).Error; err != nil {
		return fmt.Errorf("failed to create payment: %w", err)
	}
	return nil
}

// GetByOrderID fetches the payment by order_id (= no_invoice).
// Returns (nil, nil) when not found so the caller can
// distinguish "absent" from a real error.
func (p *paymentRepository) GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	var payment domain.Payment
	err := p.db.WithContext(ctx).Where("order_id = ?", orderID).First(&payment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}
	return &payment, nil
}

// Update saves changes to the payment status/attributes (full save).
func (p *paymentRepository) Update(ctx context.Context, payment *domain.Payment) error {
	if err := p.db.WithContext(ctx).Save(payment).Error; err != nil {
		return fmt.Errorf("failed to update payment: %w", err)
	}
	return nil
}

// ListStalePending implements [domain.PaymentRepository]. Pending payments past
// expiry_time (+10 min grace, giving a normal webhook time to arrive) or —
// if expiry is not recorded — older than 24h, are picked for reconciliation.
func (p *paymentRepository) ListStalePending(ctx context.Context, limit int) ([]*domain.Payment, error) {
	now := time.Now()
	var payments []*domain.Payment
	err := p.db.WithContext(ctx).
		Where("status = ?", domain.PaymentPending).
		Where("(expiry_time IS NOT NULL AND expiry_time < ?) OR (expiry_time IS NULL AND created_at < ?)",
			now.Add(-10*time.Minute), now.Add(-24*time.Hour)).
		Order("created_at ASC").
		Limit(limit).
		Find(&payments).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list stale pending payments: %w", err)
	}
	return payments, nil
}

// UpdateWithLock implements [domain.PaymentRepository]. The payment row is locked
// FOR UPDATE during the transaction so a concurrent webhook for the same order
// waits, then sees the final status via the re-check in fn.
//
// Note: fn may open another DB connection (e.g. creating a sales transaction
// via another usecase); the connection pool must be > 1 to avoid mutual waiting.
func (p *paymentRepository) UpdateWithLock(ctx context.Context, orderID string, fn func(payment *domain.Payment) (bool, error)) error {
	return runTxDB(ctx, p.db, func(tx *gorm.DB) error {
		var payment domain.Payment
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("order_id = ?", orderID).First(&payment).Error; err != nil {
			return fmt.Errorf("failed to lock payment: %w", err)
		}
		save, err := fn(&payment)
		if err != nil {
			return err
		}
		if !save {
			return nil
		}
		if err := tx.Save(&payment).Error; err != nil {
			return fmt.Errorf("failed to update payment: %w", err)
		}
		return nil
	})
}
