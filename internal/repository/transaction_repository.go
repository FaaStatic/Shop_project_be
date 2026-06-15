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
// Mengurangi stok tiap produk (dengan row lock), meng-upsert hutang pelanggan
// bila pembayaran hutang, lalu menyimpan transaksi — semuanya dalam satu
// transaksi DB sehingga berhasil/dibatalkan bersama.
func (t *transactionRepository) CreateTransaction(ctx context.Context, transaction *domain.Transactions, isHutang bool) error {
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Kurangi stok tiap produk dengan lock agar tidak balapan stok.
		for _, d := range transaction.TransactionDetail {
			var product domain.Products
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("id = ?", d.ProductID).First(&product).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fmt.Errorf("product with id %s not found", d.ProductID)
				}
				return fmt.Errorf("failed to lock product: %w", err)
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

		// 2. Untuk hutang: buat hutang baru atau tambah ke hutang pelanggan yang ada.
		if isHutang && transaction.CustomerID != nil {
			var debt domain.Debts
			err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("customer_id = ?", *transaction.CustomerID).First(&debt).Error
			switch {
			case errors.Is(err, gorm.ErrRecordNotFound):

				totalDebt := debt.TotalDebt + transaction.TotalTransaction
				debt = domain.Debts{
					CustomerID:    *transaction.CustomerID,
					TotalDebt:     totalDebt,
					RemainingDebt: debt.RemainingDebt,
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

		// 3. Simpan transaksi beserta detailnya.
		if err := tx.Session(&gorm.Session{FullSaveAssociations: true}).Create(transaction).Error; err != nil {
			return fmt.Errorf("failed to create transaction: %w", err)
		}
		return nil
	})
}

// DeleteTransaction implements [domain.TransactionRepository].
// Mengembalikan stok tiap produk sebesar qty yang terjual, lalu menghapus
// transaksinya — semuanya dalam satu transaksi DB sehingga konsisten.
func (t *transactionRepository) DeleteTransaction(ctx context.Context, id uuid.UUID) error {
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Ambil transaksi beserta detailnya (butuh product_id & qty per item).
		var trx domain.Transactions
		if err := tx.Preload("TransactionDetail").Where("id = ?", id).First(&trx).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("transaction with id %s not found", id)
			}
			return fmt.Errorf("failed to get transaction: %w", err)
		}

		// 2. Kembalikan stok tiap produk dengan row lock agar tidak balapan.
		for _, d := range trx.TransactionDetail {
			var product domain.Products
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("id = ?", d.ProductID).First(&product).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fmt.Errorf("product with id %s not found", d.ProductID)
				}
				return fmt.Errorf("failed to lock product: %w", err)
			}
			if err := tx.Model(&domain.Products{}).Where("id = ?", d.ProductID).
				Update("stock", product.Stock+d.Qty).Error; err != nil {
				return fmt.Errorf("failed to restore product stock: %w", err)
			}
		}

		// 3. Bila transaksi terhubung ke hutang, kurangi saldo hutang pelanggan
		//    sebesar nilai transaksi ini (clamp di 0 agar tidak negatif).
		if trx.DebtID != nil {
			var debt domain.Debts
			err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("id = ?", *trx.DebtID).First(&debt).Error
			switch {
			case errors.Is(err, gorm.ErrRecordNotFound):
				// Hutang sudah tidak ada; tidak ada yang perlu dibalik.
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

		// 4. Hapus transaksinya (soft delete).
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

// GetMonthlyReport implements [domain.TransactionRepository].
// Mengagregasi jumlah transaksi dan nilai transaksi pada bulan & tahun tertentu.
// payment_type = 1 (hutang) dipisahkan dari pendapatan karena belum diterima.
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
// Mengagregasi transaksi per hari (per tanggal) dalam bulan & tahun tertentu,
// diurut menaik berdasarkan tanggal. Hanya hari yang ada transaksi yang muncul.
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
// Rekap produk terjual selama sebulan (total qty & total penjualan per produk),
// diurut dari yang paling banyak terjual.
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
// Rekap produk terjual per hari (qty & penjualan per produk pada tiap tanggal),
// diurut menaik per tanggal lalu produk terlaris di hari itu.
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
		return fmt.Errorf("failed to update product: %w", result.Error)
	}
	return nil
}
