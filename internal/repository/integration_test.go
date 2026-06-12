package repository

import (
	"context"
	"errors"
	"os"
	"testing"

	"shop_project_be/internal/domain"
	"shop_project_be/pkg/dbtx"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Integration test ini menyentuh PostgreSQL sungguhan (butuh fitur seperti
// SELECT ... FOR UPDATE dan unique constraint yang tidak ada di SQLite). Test
// di-SKIP otomatis bila TEST_DATABASE_URL kosong, jadi `go test ./...` biasa
// tetap aman. Set ke database TEST KHUSUS (jangan production) untuk menjalankan:
//
//	TEST_DATABASE_URL="host=localhost user=... password=... dbname=shop_test port=5432 sslmode=disable" \
//	  go test ./internal/repository/ -run Integration -v
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL kosong; lewati integration test (butuh DB test khusus)")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		// TranslateError perlu agar unique violation jadi gorm.ErrDuplicatedKey
		// (sama seperti konfigurasi produksi).
		TranslateError: true,
		// Hindari pembuatan FK constraint agar test bisa insert dengan id acak.
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("gagal koneksi DB test: %v", err)
	}
	if err := db.AutoMigrate(
		&domain.Users{}, &domain.Customers{}, &domain.Debts{},
		&domain.Products{}, &domain.Transactions{}, &domain.TransactionsDetail{},
	); err != nil {
		t.Fatalf("gagal migrate: %v", err)
	}
	return db
}

// Membuktikan bahwa bila langkah di dalam TxManager.Do gagal, perubahan stok
// (yang dilakukan lewat UpdateStockWithLock) ikut di-rollback.
func TestIntegration_SaleRollsBackStock(t *testing.T) {
	db := setupTestDB(t)
	prodRepo := NewProductRepository(db)
	mgr := dbtx.NewManager(db)
	ctx := context.Background()

	p := &domain.Products{
		SKU:          "IT-" + uuid.NewString()[:8],
		ProductName:  "Produk Integration",
		SellingPrice: 1000,
		Stock:        5,
	}
	if err := prodRepo.AddProduct(ctx, p); err != nil {
		t.Fatalf("add product: %v", err)
	}
	t.Cleanup(func() { db.Unscoped().Delete(&domain.Products{}, "id = ?", p.ID) })

	boom := errors.New("paksa rollback")
	err := mgr.Do(ctx, func(ctx context.Context) error {
		if err := prodRepo.UpdateStockWithLock(ctx, p.ID, -3); err != nil {
			return err
		}
		return boom // gagal setelah stok dikurangi -> harus rollback
	})
	if !errors.Is(err, boom) {
		t.Fatalf("expected boom error, got %v", err)
	}

	var after domain.Products
	if err := db.First(&after, "id = ?", p.ID).Error; err != nil {
		t.Fatalf("reload product: %v", err)
	}
	if after.Stock != 5 {
		t.Fatalf("stok = %d, harusnya tetap 5 (rollback)", after.Stock)
	}
}

// Membuktikan invoice kembar terdeteksi sebagai domain.ErrDuplicateInvoice
// (lewat unique constraint DB), bukan error mentah.
func TestIntegration_DuplicateInvoice(t *testing.T) {
	db := setupTestDB(t)
	trxRepo := NewTransactionRepository(db)
	ctx := context.Background()

	inv := "IT-INV-" + uuid.NewString()[:8]
	userID := uuid.New()
	newTrx := func() *domain.Transactions {
		return &domain.Transactions{
			NoInvoice:        inv,
			UserID:           userID,
			PaymentType:      0, // tunai
			TotalTransaction: 1000,
		}
	}

	if err := trxRepo.CreateTransaction(ctx, newTrx()); err != nil {
		t.Fatalf("create pertama: %v", err)
	}
	t.Cleanup(func() { db.Unscoped().Where("no_invoice = ?", inv).Delete(&domain.Transactions{}) })

	err := trxRepo.CreateTransaction(ctx, newTrx())
	if !errors.Is(err, domain.ErrDuplicateInvoice) {
		t.Fatalf("expected domain.ErrDuplicateInvoice, got %v", err)
	}
}
