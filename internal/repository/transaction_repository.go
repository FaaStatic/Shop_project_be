package repository

import (
	"context"
	"errors"
	"fmt"
	"shop_project_be/internal/constant/enum"
	"shop_project_be/internal/constant/paginated"
	"shop_project_be/internal/domain"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type transactionRepository struct {
	db *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) domain.TransactionRepository {
	return &transactionRepository{db: db}
}

// CheckTransactionByNoInvoice implements [domain.TransactionRepository].
func (t *transactionRepository) CheckTransactionByNoInvoice(ctx context.Context, noInvoice string) (*domain.Transactions, error) {
	var item domain.Transactions
	result := t.db.WithContext(ctx).Where("no_invoice = ?", noInvoice).First(&item)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get transaction: %w", result.Error)
	}
	return &item, nil
}

// CreateTransactionAtomic implements [domain.TransactionRepository].
// Decrements each product's stock (with a row lock), upserts the customer's debt
// if it's a debt payment, then saves the transaction — all in one
// DB transaction so they succeed/roll back together.
func (t *transactionRepository) CreateTransaction(ctx context.Context, transaction *domain.Transactions, isHutang bool, deductStock bool) error {
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Decrement each product's stock with a lock to avoid a stock race.
		// deductStock is false for transactions from online payments: their stock
		// was already deducted at charge reservation, don't deduct it twice.
		if deductStock {
			// Lock all products up-front in a deterministic id order so
			// two concurrent sales of the same products (different item order)
			// don't deadlock. The loop below stays in the original order -> error messages
			// are unchanged.
			ids := make([]uuid.UUID, 0, len(transaction.TransactionDetail))
			for _, d := range transaction.TransactionDetail {
				ids = append(ids, d.ProductID)
			}
			if err := lockProductsOrdered(tx, ids); err != nil {
				return err
			}
			for _, d := range transaction.TransactionDetail {
				var product domain.Products
				if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
					Where("id = ?", d.ProductID).First(&product).Error; err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						return fmt.Errorf("product with id %s not found", d.ProductID)
					}
					return fmt.Errorf("failed to lock product: %w", err)
				}
				if product.ProductType.IsDigital() {
					continue // digital goods are not stock-managed
				}
				qty := d.Qty
				if product.Stock < qty {
					return fmt.Errorf("insufficient stock for product %s (current: %v, requested: %v)", d.ProductID, product.Stock, qty)
				}
				if err := tx.Model(&domain.Products{}).Where("id = ?", d.ProductID).
					Update("stock", product.Stock-qty).Error; err != nil {
					return fmt.Errorf("failed to update product stock: %w", err)
				}
			}
		}

		// 2. For debt: create a new debt or add to the customer's existing debt.
		if isHutang && transaction.CustomerID != nil {
			var debt domain.Debts
			err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("customer_id = ?", *transaction.CustomerID).First(&debt).Error
			switch {
			case errors.Is(err, gorm.ErrRecordNotFound):

				totalDebt := debt.TotalDebt + transaction.TotalTransaction
				debt = domain.Debts{
					CustomerID:    *transaction.CustomerID,
					TotalDebt:     totalDebt,
					RemainingDebt: totalDebt,
					Status:        enum.BELUM_LUNAS,
				}
				if err := tx.Create(&debt).Error; err != nil {
					return fmt.Errorf("failed to create debt: %w", err)
				}
			case err != nil:
				return fmt.Errorf("failed to get debt: %w", err)
			default:
				if err := tx.Model(&domain.Debts{}).Where("id = ?", debt.ID).
					Updates(map[string]interface{}{
						"total_debt":     debt.TotalDebt + transaction.TotalTransaction,
						"remaining_debt": debt.RemainingDebt + transaction.TotalTransaction,
						"status":         enum.BELUM_LUNAS,
					}).Error; err != nil {
					return fmt.Errorf("failed to update debt: %w", err)
				}
			}
			transaction.DebtID = &debt.ID
		}

		// 3. Save the transaction along with its details.
		if err := tx.Session(&gorm.Session{FullSaveAssociations: true}).Create(transaction).Error; err != nil {
			return fmt.Errorf("failed to create transaction: %w", err)
		}
		return nil
	})
}

// DeleteTransaction implements [domain.TransactionRepository].
// Returns each product's stock by the sold qty, then deletes
// the transaction — all in one DB transaction for consistency.
func (t *transactionRepository) DeleteTransaction(ctx context.Context, id uuid.UUID) error {
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Fetch the transaction along with its details (needs product_id & qty per item).
		var trx domain.Transactions
		if err := tx.Preload("TransactionDetail").Where("id = ?", id).First(&trx).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("transaction with id %s not found", id)
			}
			return fmt.Errorf("failed to get transaction: %w", err)
		}

		// 2. Return each product's stock with a row lock to avoid a race.
		// Lock up-front in a deterministic id order -> preventing deadlock
		// if two concurrent deletions share products.
		restoreIDs := make([]uuid.UUID, 0, len(trx.TransactionDetail))
		for _, d := range trx.TransactionDetail {
			restoreIDs = append(restoreIDs, d.ProductID)
		}
		if err := lockProductsOrdered(tx, restoreIDs); err != nil {
			return err
		}
		for _, d := range trx.TransactionDetail {
			var product domain.Products
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("id = ?", d.ProductID).First(&product).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fmt.Errorf("product with id %s not found", d.ProductID)
				}
				return fmt.Errorf("failed to lock product: %w", err)
			}
			if product.ProductType.IsDigital() {
				continue
			}
			if err := tx.Model(&domain.Products{}).Where("id = ?", d.ProductID).
				Update("stock", product.Stock+d.Qty).Error; err != nil {
				return fmt.Errorf("failed to restore product stock: %w", err)
			}
		}

		// 3. If the transaction is linked to a debt, reduce the customer's debt balance
		//    by this transaction's value (clamped at 0 so it can't go negative).
		if trx.DebtID != nil {
			var debt domain.Debts
			err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("id = ?", *trx.DebtID).First(&debt).Error
			switch {
			case errors.Is(err, gorm.ErrRecordNotFound):
				// The debt no longer exists; nothing to reverse.
			case err != nil:
				return fmt.Errorf("failed to lock debt: %w", err)
			default:
				newTotal := debt.TotalDebt - trx.TotalTransaction
				if newTotal < 0 {
					newTotal = 0
				}
				newRemaining := debt.RemainingDebt - trx.TotalTransaction
				if newRemaining < 0 {
					newRemaining = 0
				}
				status := enum.BELUM_LUNAS
				if newRemaining <= 0 {
					status = enum.LUNAS
				}
				if err := tx.Model(&domain.Debts{}).Where("id = ?", debt.ID).
					Updates(map[string]interface{}{
						"total_debt":     newTotal,
						"remaining_debt": newRemaining,
						"status":         status,
					}).Error; err != nil {
					return fmt.Errorf("failed to reverse debt: %w", err)
				}
			}
		}

		// 4. Delete the transaction (soft delete).
		result := tx.Where("id = ?", id).Delete(&domain.Transactions{})
		if result.Error != nil {
			return fmt.Errorf("failed to delete transaction: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("transaction with id %s not found", id)
		}
		return nil
	})
}

// GetAllTransaction implements [domain.TransactionRepository].
func (t *transactionRepository) GetAllTransaction(ctx context.Context, filter domain.FilterTransaction) (*domain.ResultTransaction, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 10
	}

	order := "DESC"
	if strings.ToUpper(filter.Order) == "ASC" {
		order = "ASC"
	}

	query := t.db.Preload("User").
		Preload("Customer").
		Preload("TransactionDetail").Preload("TransactionDetail.Product").WithContext(ctx).Model(&domain.Transactions{})

	// Filters are applied before the cursor for cross-page consistency: the cursor may
	// only move within the already-filtered result, not the whole table.
	if filter.NoInvoices != "" {
		escaped := strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(filter.NoInvoices)
		query = query.Where("no_invoice LIKE ? ESCAPE '\\'", "%"+escaped+"%")
	}
	if filter.TypeTrx != nil {
		query = query.Where("payment_type = ?", *filter.TypeTrx)
	}
	if filter.DateStart != nil && *filter.DateStart != "" {
		query = query.Where("created_at >= ?", *filter.DateStart)
	}
	if filter.DateEnd != nil && *filter.DateEnd != "" {
		query = query.Where("created_at <= ?", *filter.DateEnd)
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

// GetMonthlyReport implements [domain.TransactionRepository].
// Aggregates the transaction count and value for a given month & year.
// payment_type = 1 (debt) is separated from revenue because it hasn't been received.
func (t *transactionRepository) GetMonthlyReport(ctx context.Context, month int, year int) (*domain.MonthlyReport, error) {
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	var report domain.MonthlyReport
	result := t.db.WithContext(ctx).
		Model(&domain.Transactions{}).
		Where("created_at >= ? AND created_at < ?", start, end).
		Select(`COUNT(*) AS total_transaction,
			COALESCE(SUM(total_transaction) FILTER (WHERE payment_type <> 1), 0) AS total_revenue,
			COALESCE(SUM(total_transaction) FILTER (WHERE payment_type = 1), 0) AS total_debt,
			COALESCE(SUM(total_transaction), 0) AS grand_total`).
		Scan(&report)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get monthly report: %w", result.Error)
	}
	return &report, nil
}

// GetDailyReport implements [domain.TransactionRepository].
// Aggregates transactions per day (per date) within a given month & year,
// sorted ascending by date. Only days with transactions appear.
func (t *transactionRepository) GetDailyReport(ctx context.Context, month int, year int) ([]domain.DailyReport, error) {
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	var rows []domain.DailyReport
	result := t.db.WithContext(ctx).
		Model(&domain.Transactions{}).
		Where("created_at >= ? AND created_at < ?", start, end).
		Select(`(created_at)::date AS date,
			COUNT(*) AS total_transaction,
			COALESCE(SUM(total_transaction) FILTER (WHERE payment_type <> 1), 0) AS total_revenue,
			COALESCE(SUM(total_transaction) FILTER (WHERE payment_type = 1), 0) AS total_debt,
			COALESCE(SUM(total_transaction), 0) AS grand_total`).
		Group(`(created_at)::date`).
		Order(`(created_at)::date ASC`).
		Scan(&rows)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get daily report: %w", result.Error)
	}
	return rows, nil
}

// GetMonthlyProductSold implements [domain.TransactionRepository].
// Recap of products sold during a month (total qty & total sales per product),
// sorted from the best-selling first.
func (t *transactionRepository) GetMonthlyProductSold(ctx context.Context, month int, year int) ([]domain.ProductSoldReport, error) {
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	var rows []domain.ProductSoldReport
	result := t.db.WithContext(ctx).
		Table("transactions_detail AS td").
		Joins("JOIN transactions AS t ON t.id = td.transaction_id").
		Joins("JOIN products AS p ON p.id = td.product_id").
		Where("t.created_at >= ? AND t.created_at < ? AND t.deleted_at IS NULL", start, end).
		Select("p.product_name AS product_name, SUM(td.qty) AS qty, SUM(td.subtotal) AS total").
		Group("p.product_name").
		Order("qty DESC").
		Scan(&rows)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get monthly product sold: %w", result.Error)
	}
	return rows, nil
}

// GetDailyProductSold implements [domain.TransactionRepository].
// Recap of products sold per day (qty & sales per product on each date),
// sorted ascending by date then the best-selling product that day.
func (t *transactionRepository) GetDailyProductSold(ctx context.Context, month int, year int) ([]domain.DailyProductSoldReport, error) {
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	var rows []domain.DailyProductSoldReport
	result := t.db.WithContext(ctx).
		Table("transactions_detail AS td").
		Joins("JOIN transactions AS t ON t.id = td.transaction_id").
		Joins("JOIN products AS p ON p.id = td.product_id").
		Where("t.created_at >= ? AND t.created_at < ? AND t.deleted_at IS NULL", start, end).
		Select("(t.created_at)::date AS date, p.product_name AS product_name, SUM(td.qty) AS qty, SUM(td.subtotal) AS total").
		Group("(t.created_at)::date, p.product_name").
		Order("(t.created_at)::date ASC, qty DESC").
		Scan(&rows)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get daily product sold: %w", result.Error)
	}
	return rows, nil
}

// GetTransactionByID implements [domain.TransactionRepository].
func (t *transactionRepository) GetTransactionByID(ctx context.Context, id uuid.UUID) (*domain.Transactions, error) {
	var item domain.Transactions
	result := t.db.WithContext(ctx).Preload("User").
		Preload("Customer").
		Preload("TransactionDetail").Preload("TransactionDetail.Product").Where("id = ?", id).First(&item)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("transaction with id %s not found: %w", id, result.Error)
		}
		return nil, fmt.Errorf("failed to get transaction: %w", result.Error)
	}
	return &item, nil
}

// UpdateTransaction implements [domain.TransactionRepository].
func (t *transactionRepository) UpdateTransaction(ctx context.Context, id uuid.UUID, trx *domain.Transactions) error {
	result := t.db.WithContext(ctx).Model(&domain.Transactions{}).Where("id = ?", id).Updates(trx)

	if result.Error != nil {
		return fmt.Errorf("failed to update transaction: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("transaction with id %s not found", id)
	}
	return nil
}
