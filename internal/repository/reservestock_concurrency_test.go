package repository_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"shop_project_be/internal/domain"
	"shop_project_be/internal/repository"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestReserveStock_Concurrent proves the stock guarantee under real concurrency:
// when N goroutines each try to reserve 1 unit of a product that has exactly
// `stock` units, precisely `stock` reservations succeed, the rest fail with
// "insufficient stock", and the row never goes negative — i.e. no oversell and
// no lost update between two simultaneous users.
//
// It needs a real (migrated) Postgres because SELECT ... FOR UPDATE is what
// provides the guarantee; it is skipped unless TEST_DATABASE_DSN is set, e.g.:
//
//	TEST_DATABASE_DSN='host=localhost user=user_test password=... dbname=db_toko port=5432 sslmode=disable' \
//	  go test ./internal/repository -run TestReserveStock_Concurrent -race -v
func TestReserveStock_Concurrent(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_DSN to run the concurrency test against a real Postgres")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql.DB: %v", err)
	}
	// The pool must comfortably exceed the goroutine count so contention happens
	// on the row lock (the thing under test), not on the connection pool.
	sqlDB.SetMaxOpenConns(30)
	defer sqlDB.Close()

	const (
		stock   = 20  // units available
		workers = 100 // simultaneous buyers, each wanting 1 unit
	)

	product := &domain.Products{
		SKU:         "CONCURRENCY-TEST-" + uuid.NewString(),
		ProductName: "concurrency test product",
		Stock:       stock,
	}
	if err := db.Create(product).Error; err != nil {
		t.Fatalf("seed product: %v", err)
	}
	t.Cleanup(func() {
		db.Unscoped().Delete(&domain.Products{}, "id = ?", product.ID)
	})

	repo := repository.NewProductRepository(db)

	var success, insufficient, other int64
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start // release all goroutines at once to maximize contention
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			err := repo.ReserveStock(ctx, []domain.PaymentItem{{ProductID: product.ID, Qty: 1}})
			switch {
			case err == nil:
				atomic.AddInt64(&success, 1)
			case strings.Contains(err.Error(), "insufficient stock"):
				atomic.AddInt64(&insufficient, 1)
			default:
				atomic.AddInt64(&other, 1)
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()

	if other != 0 {
		t.Fatalf("got %d unexpected errors (deadlock/serialization should have been retried, not surfaced)", other)
	}
	if success != stock {
		t.Errorf("successful reservations = %d, want exactly %d (oversell or lost reservation)", success, stock)
	}
	if insufficient != workers-stock {
		t.Errorf("insufficient-stock rejections = %d, want %d", insufficient, workers-stock)
	}

	var final domain.Products
	if err := db.First(&final, "id = ?", product.ID).Error; err != nil {
		t.Fatalf("reload product: %v", err)
	}
	if final.Stock != 0 {
		t.Errorf("final stock = %v, want 0 (must never oversell or go negative)", final.Stock)
	}
	fmt.Printf("reserved=%d rejected=%d final_stock=%v\n", success, insufficient, final.Stock)
}
