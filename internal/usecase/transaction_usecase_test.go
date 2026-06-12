package usecase

import (
	"context"
	"fmt"
	"testing"

	"shop_project_be/internal/constant/enum"
	"shop_project_be/internal/domain"
	requestdto "shop_project_be/internal/dto/request_dto"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Fakes: implementasi in-memory dari interface repository + TxManager.
// Hanya method yang dipakai AddTransaction yang berisi logika; sisanya stub.
// ---------------------------------------------------------------------------

// passthroughTx menjalankan fn langsung (tanpa DB). Cukup untuk menguji
// ORKESTRASI AddTransaction. Atomicity/rollback sebenarnya butuh DB nyata.
type passthroughTx struct{}

func (passthroughTx) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

type fakeTrxRepo struct {
	existingInvoice *domain.Transactions // dikembalikan CheckTransactionByNoInvoice
	created         *domain.Transactions // hasil CreateTransaction terakhir
	createErr       error
}

func (f *fakeTrxRepo) CheckTransactionByNoInvoice(ctx context.Context, noInvoice string) (*domain.Transactions, error) {
	return f.existingInvoice, nil
}
func (f *fakeTrxRepo) CreateTransaction(ctx context.Context, trx *domain.Transactions) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.created = trx
	return nil
}
func (f *fakeTrxRepo) GetTransactionByID(ctx context.Context, id uuid.UUID) (*domain.Transactions, error) {
	return nil, nil
}
func (f *fakeTrxRepo) GetAllTransaction(ctx context.Context, filter domain.FilterTransaction) (*domain.ResultTransaction, error) {
	return nil, nil
}
func (f *fakeTrxRepo) DeleteTransaction(ctx context.Context, id uuid.UUID) error { return nil }
func (f *fakeTrxRepo) UpdateTransaction(ctx context.Context, id uuid.UUID, trx *domain.Transactions) error {
	return nil
}
func (f *fakeTrxRepo) GetMonthlyReport(ctx context.Context, m, y int) (*domain.MonthlyReport, error) {
	return nil, nil
}
func (f *fakeTrxRepo) GetDailyReport(ctx context.Context, m, y int) ([]domain.DailyReport, error) {
	return nil, nil
}
func (f *fakeTrxRepo) GetMonthlyProductSold(ctx context.Context, m, y int) ([]domain.ProductSoldReport, error) {
	return nil, nil
}
func (f *fakeTrxRepo) GetDailyProductSold(ctx context.Context, m, y int) ([]domain.DailyProductSoldReport, error) {
	return nil, nil
}

type fakeProductRepo struct {
	products map[uuid.UUID]*domain.Products
}

func (f *fakeProductRepo) GetProduct(ctx context.Context, id uuid.UUID) (*domain.Products, error) {
	p, ok := f.products[id]
	if !ok {
		return nil, fmt.Errorf("product not found")
	}
	return p, nil
}
func (f *fakeProductRepo) UpdateStockWithLock(ctx context.Context, id uuid.UUID, delta int) error {
	p, ok := f.products[id]
	if !ok {
		return fmt.Errorf("product not found")
	}
	newStock := p.Stock + delta
	if newStock < 0 {
		return fmt.Errorf("stok tidak cukup untuk produk %s", p.SKU)
	}
	p.Stock = newStock
	return nil
}
func (f *fakeProductRepo) AddProduct(ctx context.Context, p *domain.Products) error { return nil }
func (f *fakeProductRepo) UpdateProduct(ctx context.Context, p *domain.Products, id uuid.UUID) error {
	return nil
}
func (f *fakeProductRepo) AddBulkProduct(ctx context.Context, ps []*domain.Products) (*domain.BulkInsertResult, error) {
	return nil, nil
}
func (f *fakeProductRepo) DeleteProduct(ctx context.Context, id uuid.UUID) error { return nil }
func (f *fakeProductRepo) GetAllProduct(ctx context.Context, filter domain.FilterAllProduct) (*domain.PaginatedItem, error) {
	return nil, nil
}
func (f *fakeProductRepo) UpdateProductWithLock(ctx context.Context, id uuid.UUID, fields map[string]interface{}, delta int) error {
	return nil
}

type fakeUserRepo struct{ user *domain.Users }

func (f *fakeUserRepo) GetUserById(ctx context.Context, id uuid.UUID) (*domain.Users, error) {
	return f.user, nil
}
func (f *fakeUserRepo) GetUserLogin(ctx context.Context, id uuid.UUID) (*domain.Users, error) {
	return f.user, nil
}
func (f *fakeUserRepo) RegisterUser(ctx context.Context, u *domain.Users) error { return nil }
func (f *fakeUserRepo) GetUserByUsername(ctx context.Context, username string) (*domain.Users, error) {
	return f.user, nil
}

type fakeCustomerRepo struct {
	debtID   *uuid.UUID // dikembalikan GetDebtIdByCustomerId
	customer *[]domain.Customers
}

func (f *fakeCustomerRepo) GetDebtIdByCustomerId(ctx context.Context, customerId uuid.UUID) (*uuid.UUID, error) {
	return f.debtID, nil
}
func (f *fakeCustomerRepo) GetCustomer(ctx context.Context, id uuid.UUID) (*[]domain.Customers, error) {
	return f.customer, nil
}
func (f *fakeCustomerRepo) LockCustomerForUpdate(ctx context.Context, id uuid.UUID) error {
	return nil
}
func (f *fakeCustomerRepo) UpdateCustomer(ctx context.Context, id uuid.UUID, c *domain.Customers) error {
	return nil
}
func (f *fakeCustomerRepo) AddCustomer(ctx context.Context, c *domain.Customers) error { return nil }
func (f *fakeCustomerRepo) DeleteCustomer(ctx context.Context, id uuid.UUID) error     { return nil }
func (f *fakeCustomerRepo) GetAllCustomer(ctx context.Context, search string, limit, offset int) ([]*domain.Customers, error) {
	return nil, nil
}

type fakeDebtRepo struct {
	debts   map[uuid.UUID]*domain.Debts
	added   *domain.Debts // hutang baru terakhir
	updated *domain.Debts // hutang ter-update terakhir
}

func (f *fakeDebtRepo) AddDebt(ctx context.Context, debt *domain.Debts) error {
	if debt.ID == uuid.Nil {
		debt.ID = uuid.New()
	}
	if f.debts == nil {
		f.debts = map[uuid.UUID]*domain.Debts{}
	}
	f.debts[debt.ID] = debt
	f.added = debt
	return nil
}
func (f *fakeDebtRepo) GetDebtByID(ctx context.Context, id uuid.UUID) (*domain.Debts, error) {
	d, ok := f.debts[id]
	if !ok {
		return nil, fmt.Errorf("debt not found")
	}
	return d, nil
}
func (f *fakeDebtRepo) UpdateDebt(ctx context.Context, id uuid.UUID, debt *domain.Debts) error {
	f.debts[id] = debt
	f.updated = debt
	return nil
}
func (f *fakeDebtRepo) DeleteDebt(ctx context.Context, id uuid.UUID) error { return nil }
func (f *fakeDebtRepo) GetAllDebt(ctx context.Context, filter domain.FilterDebt) (*domain.DebtsPaginated, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newProduct(price float64, stock int) *domain.Products {
	return &domain.Products{
		ID:               uuid.New(),
		SKU:              "SKU-TEST",
		ProductName:      "Beras",
		SellingPrice:     price,
		SellingPriceDebt: price + 1000,
		Stock:            stock,
	}
}

func buildUsecase(trx *fakeTrxRepo, prod *fakeProductRepo, cust *fakeCustomerRepo, debt *fakeDebtRepo) domain.TransactionUsecase {
	user := &domain.Users{ID: uuid.New(), Username: "kasir"}
	return NewTransactionUsecase(trx, prod, &fakeUserRepo{user: user}, cust, debt, passthroughTx{}, "Toko Ibu", zap.NewNop())
}

func customerFound() *[]domain.Customers {
	return &[]domain.Customers{{ID: uuid.New(), Name: "Budi"}}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// Penjualan tunai: stok berkurang, total & subtotal dihitung server-side dari
// harga produk, dan nilai total/subtotal dari client diabaikan.
func TestAddTransaction_Tunai_ComputesTotalAndDecrementsStock(t *testing.T) {
	prod := newProduct(10000, 10)
	prodRepo := &fakeProductRepo{products: map[uuid.UUID]*domain.Products{prod.ID: prod}}
	trxRepo := &fakeTrxRepo{}
	uc := buildUsecase(trxRepo, prodRepo, &fakeCustomerRepo{}, &fakeDebtRepo{})

	uid := uuid.New().String()
	err := uc.AddTransaction(context.Background(), &requestdto.AddTransactionRequest{
		NoInvoice:        "INV-1",
		TypePayment:      "tunai",
		TotalTransaction: 1, // bogus, harus diabaikan
		UserId:           uid,
		Details: []requestdto.AddTransactionDetailRequest{
			{ProductId: prod.ID.String(), Qty: 3, Subtotal: 999999}, // bogus subtotal
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if trxRepo.created == nil {
		t.Fatal("transaction was not created")
	}
	if trxRepo.created.TotalTransaction != 30000 {
		t.Fatalf("total = %v, want 30000", trxRepo.created.TotalTransaction)
	}
	d := trxRepo.created.TransactionDetail[0]
	if d.Price != 10000 || d.Subtotal != 30000 {
		t.Fatalf("detail price=%v subtotal=%v, want 10000/30000", d.Price, d.Subtotal)
	}
	if prod.Stock != 7 {
		t.Fatalf("stock = %d, want 7", prod.Stock)
	}
	if trxRepo.created.DebtID != nil {
		t.Fatal("tunai should not create debt")
	}
}

// Stok tidak cukup -> transaksi gagal & tidak ada yang ter-insert.
func TestAddTransaction_InsufficientStock(t *testing.T) {
	prod := newProduct(10000, 2)
	prodRepo := &fakeProductRepo{products: map[uuid.UUID]*domain.Products{prod.ID: prod}}
	trxRepo := &fakeTrxRepo{}
	uc := buildUsecase(trxRepo, prodRepo, &fakeCustomerRepo{}, &fakeDebtRepo{})

	err := uc.AddTransaction(context.Background(), &requestdto.AddTransactionRequest{
		NoInvoice:   "INV-2",
		TypePayment: "tunai",
		UserId:      uuid.New().String(),
		Details: []requestdto.AddTransactionDetailRequest{
			{ProductId: prod.ID.String(), Qty: 5},
		},
	})
	if err == nil {
		t.Fatal("expected error for insufficient stock")
	}
	if trxRepo.created != nil {
		t.Fatal("transaction must not be created when stock is insufficient")
	}
}

// Hutang untuk customer yang belum punya hutang -> buat hutang baru dengan
// TotalDebt == RemainingDebt == total.
func TestAddTransaction_Hutang_NewDebt(t *testing.T) {
	prod := newProduct(10000, 10)
	prodRepo := &fakeProductRepo{products: map[uuid.UUID]*domain.Products{prod.ID: prod}}
	trxRepo := &fakeTrxRepo{}
	custRepo := &fakeCustomerRepo{debtID: nil, customer: customerFound()}
	debtRepo := &fakeDebtRepo{}
	uc := buildUsecase(trxRepo, prodRepo, custRepo, debtRepo)

	cid := uuid.New().String()
	err := uc.AddTransaction(context.Background(), &requestdto.AddTransactionRequest{
		NoInvoice:   "INV-3",
		TypePayment: "hutang",
		UserId:      uuid.New().String(),
		CustomerId:  &cid,
		Details: []requestdto.AddTransactionDetailRequest{
			{ProductId: prod.ID.String(), Qty: 2}, // total 20000
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if debtRepo.added == nil {
		t.Fatal("new debt was not created")
	}
	if debtRepo.added.TotalDebt != 20000 || debtRepo.added.RemainingDebt != 20000 {
		t.Fatalf("debt total=%v remaining=%v, want 20000/20000", debtRepo.added.TotalDebt, debtRepo.added.RemainingDebt)
	}
	if trxRepo.created.DebtID == nil || *trxRepo.created.DebtID != debtRepo.added.ID {
		t.Fatal("transaction.DebtID not linked to created debt")
	}
}

// Hutang untuk customer yang sudah punya hutang -> akumulasi: TotalDebt dan
// RemainingDebt sama-sama bertambah.
func TestAddTransaction_Hutang_ExistingDebtAccumulates(t *testing.T) {
	prod := newProduct(10000, 10)
	prodRepo := &fakeProductRepo{products: map[uuid.UUID]*domain.Products{prod.ID: prod}}
	trxRepo := &fakeTrxRepo{}

	existingID := uuid.New()
	debtRepo := &fakeDebtRepo{debts: map[uuid.UUID]*domain.Debts{
		existingID: {ID: existingID, TotalDebt: 5000, RemainingDebt: 5000, Status: enum.BELUM_LUNAS},
	}}
	custRepo := &fakeCustomerRepo{debtID: &existingID, customer: customerFound()}
	uc := buildUsecase(trxRepo, prodRepo, custRepo, debtRepo)

	cid := uuid.New().String()
	err := uc.AddTransaction(context.Background(), &requestdto.AddTransactionRequest{
		NoInvoice:   "INV-4",
		TypePayment: "hutang",
		UserId:      uuid.New().String(),
		CustomerId:  &cid,
		Details: []requestdto.AddTransactionDetailRequest{
			{ProductId: prod.ID.String(), Qty: 3}, // total 30000
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if debtRepo.updated == nil {
		t.Fatal("existing debt was not updated")
	}
	if debtRepo.updated.TotalDebt != 35000 || debtRepo.updated.RemainingDebt != 35000 {
		t.Fatalf("debt total=%v remaining=%v, want 35000/35000", debtRepo.updated.TotalDebt, debtRepo.updated.RemainingDebt)
	}
}

// Nomor invoice duplikat -> ditolak sebelum menyentuh stok.
func TestAddTransaction_DuplicateInvoice(t *testing.T) {
	prod := newProduct(10000, 10)
	prodRepo := &fakeProductRepo{products: map[uuid.UUID]*domain.Products{prod.ID: prod}}
	trxRepo := &fakeTrxRepo{existingInvoice: &domain.Transactions{NoInvoice: "INV-5"}}
	uc := buildUsecase(trxRepo, prodRepo, &fakeCustomerRepo{}, &fakeDebtRepo{})

	err := uc.AddTransaction(context.Background(), &requestdto.AddTransactionRequest{
		NoInvoice:   "INV-5",
		TypePayment: "tunai",
		UserId:      uuid.New().String(),
		Details:     []requestdto.AddTransactionDetailRequest{{ProductId: prod.ID.String(), Qty: 1}},
	})
	if err == nil {
		t.Fatal("expected duplicate invoice error")
	}
	if prod.Stock != 10 {
		t.Fatalf("stock must be untouched on duplicate invoice, got %d", prod.Stock)
	}
}

// Hutang tanpa customer_id -> ditolak.
func TestAddTransaction_Hutang_RequiresCustomer(t *testing.T) {
	prod := newProduct(10000, 10)
	prodRepo := &fakeProductRepo{products: map[uuid.UUID]*domain.Products{prod.ID: prod}}
	uc := buildUsecase(&fakeTrxRepo{}, prodRepo, &fakeCustomerRepo{}, &fakeDebtRepo{})

	err := uc.AddTransaction(context.Background(), &requestdto.AddTransactionRequest{
		NoInvoice:   "INV-6",
		TypePayment: "hutang",
		UserId:      uuid.New().String(),
		Details:     []requestdto.AddTransactionDetailRequest{{ProductId: prod.ID.String(), Qty: 1}},
	})
	if err == nil {
		t.Fatal("expected error when hutang has no customer_id")
	}
}

// Qty pecahan -> ditolak (stok berupa bilangan bulat).
func TestAddTransaction_NonIntegerQty(t *testing.T) {
	prod := newProduct(10000, 10)
	prodRepo := &fakeProductRepo{products: map[uuid.UUID]*domain.Products{prod.ID: prod}}
	trxRepo := &fakeTrxRepo{}
	uc := buildUsecase(trxRepo, prodRepo, &fakeCustomerRepo{}, &fakeDebtRepo{})

	err := uc.AddTransaction(context.Background(), &requestdto.AddTransactionRequest{
		NoInvoice:   "INV-7",
		TypePayment: "tunai",
		UserId:      uuid.New().String(),
		Details:     []requestdto.AddTransactionDetailRequest{{ProductId: prod.ID.String(), Qty: 1.5}},
	})
	if err == nil {
		t.Fatal("expected error for non-integer qty")
	}
	if trxRepo.created != nil {
		t.Fatal("transaction must not be created for invalid qty")
	}
}
