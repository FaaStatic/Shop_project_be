package repository

import (
	"context"
	"os"
	"testing"

	"shop_project_be/internal/domain"

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

// seedProduct menambah produk dengan stok awal dan mendaftarkan cleanup-nya.
func seedProduct(t *testing.T, db *gorm.DB, stock float64) *domain.Products {
	t.Helper()
	p := &domain.Products{
		SKU:          "IT-" + uuid.NewString()[:8],
		ProductName:  "Produk Integration",
		SellingPrice: 1000,
		Stock:        stock,
	}
	if err := db.Create(p).Error; err != nil {
		t.Fatalf("add product: %v", err)
	}
	t.Cleanup(func() { db.Unscoped().Delete(&domain.Products{}, "id = ?", p.ID) })
	return p
}

func saleTrx(inv string, userID, productID uuid.UUID, qty float64) *domain.Transactions {
	return &domain.Transactions{
		NoInvoice:        inv,
		UserID:           userID,
		PaymentType:      0, // tunai
		TotalTransaction: 1000 * qty,
		TransactionDetail: []domain.TransactionsDetail{
			{ProductID: productID, Price: 1000, Qty: qty, Subtotal: 1000 * qty},
		},
	}
}

// CreateTransaction harus mengurangi stok produk secara atomik di dalam satu
// transaksi DB.
func TestIntegration_CreateTransaction_DecrementsStock(t *testing.T) {
	db := setupTestDB(t)
	trxRepo := NewTransactionRepository(db)
	ctx := context.Background()

	prod := seedProduct(t, db, 5)
	inv := "IT-INV-" + uuid.NewString()[:8]
	t.Cleanup(func() { db.Unscoped().Where("no_invoice = ?", inv).Delete(&domain.Transactions{}) })

	if err := trxRepo.CreateTransaction(ctx, saleTrx(inv, uuid.New(), prod.ID, 3), false); err != nil {
		t.Fatalf("create transaction: %v", err)
	}

	var after domain.Products
	if err := db.First(&after, "id = ?", prod.ID).Error; err != nil {
		t.Fatalf("reload product: %v", err)
	}
	if after.Stock != 2 {
		t.Fatalf("stok = %v, want 2 (5 - 3)", after.Stock)
	}
}

// Stok tidak cukup -> seluruh transaksi gagal dan stok tetap utuh (rollback).
func TestIntegration_CreateTransaction_InsufficientStockRollsBack(t *testing.T) {
	db := setupTestDB(t)
	trxRepo := NewTransactionRepository(db)
	ctx := context.Background()

	prod := seedProduct(t, db, 2)
	inv := "IT-INV-" + uuid.NewString()[:8]
	t.Cleanup(func() { db.Unscoped().Where("no_invoice = ?", inv).Delete(&domain.Transactions{}) })

	err := trxRepo.CreateTransaction(ctx, saleTrx(inv, uuid.New(), prod.ID, 5), false)
	if err == nil {
		t.Fatal("expected error for insufficient stock")
	}

	var after domain.Products
	if err := db.First(&after, "id = ?", prod.ID).Error; err != nil {
		t.Fatalf("reload product: %v", err)
	}
	if after.Stock != 2 {
		t.Fatalf("stok = %v, harusnya tetap 2 (rollback)", after.Stock)
	}
}

// DeleteTransaction harus mengembalikan stok produk sebesar qty yang terjual.
func TestIntegration_DeleteTransaction_RestoresStock(t *testing.T) {
	db := setupTestDB(t)
	trxRepo := NewTransactionRepository(db)
	ctx := context.Background()

	prod := seedProduct(t, db, 5)
	inv := "IT-INV-" + uuid.NewString()[:8]
	trx := saleTrx(inv, uuid.New(), prod.ID, 3)
	t.Cleanup(func() { db.Unscoped().Where("no_invoice = ?", inv).Delete(&domain.Transactions{}) })

	if err := trxRepo.CreateTransaction(ctx, trx, false); err != nil {
		t.Fatalf("create transaction: %v", err)
	}
	if err := trxRepo.DeleteTransaction(ctx, trx.ID); err != nil {
		t.Fatalf("delete transaction: %v", err)
	}

	var after domain.Products
	if err := db.First(&after, "id = ?", prod.ID).Error; err != nil {
		t.Fatalf("reload product: %v", err)
	}
	if after.Stock != 5 {
		t.Fatalf("stok = %v, want 5 (dikembalikan setelah delete)", after.Stock)
	}
}
