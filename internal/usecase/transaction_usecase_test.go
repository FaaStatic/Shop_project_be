package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"shop_project_be/internal/domain"
	requestdto "shop_project_be/internal/dto/request_dto"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Fakes: implementasi in-memory dari interface repository.
// Hanya method yang dipakai AddTransaction yang berisi logika; sisanya stub.
//
// Catatan desain: pengurangan stok & upsert hutang dilakukan secara atomik di
// dalam TransactionRepository.CreateTransaction (satu transaksi DB), BUKAN di
// usecase. Karena itu test ini fokus pada orkestrasi usecase: validasi, hitung
// total server-side, flag isHutang, dan propagasi error dari repo.
// ---------------------------------------------------------------------------

type fakeTrxRepo struct {
	existingInvoice *domain.Transactions // dikembalikan CheckTransactionByNoInvoice
	created         *domain.Transactions // hasil CreateTransaction terakhir
	lastIsHutang    bool                 // nilai isHutang pada CreateTransaction terakhir
	createErr       error                // bila diset, CreateTransaction gagal
}

func (f *fakeTrxRepo) CheckTransactionByNoInvoice(ctx context.Context, noInvoice string) (*domain.Transactions, error) {
	return f.existingInvoice, nil
}
func (f *fakeTrxRepo) CreateTransaction(ctx context.Context, trx *domain.Transactions, isHutang bool) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.created = trx
	f.lastIsHutang = isHutang
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
func (f *fakeProductRepo) UpdateStockWithLock(ctx context.Context, id uuid.UUID, delta float64) error {
	return nil
}
func (f *fakeProductRepo) UpdateProductWithLock(ctx context.Context, id uuid.UUID, fields map[string]interface{}, delta float64) error {
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
	debtID   *uuid.UUID            // dikembalikan GetDebtIdByCustomerId
	customer *[]domain.Customers   // dikembalikan GetCustomer (nil = tidak ditemukan)
}

func (f *fakeCustomerRepo) GetDebtIdByCustomerId(ctx context.Context, customerId uuid.UUID) (*uuid.UUID, error) {
	return f.debtID, nil
}
func (f *fakeCustomerRepo) GetCustomer(ctx context.Context, id uuid.UUID) (*[]domain.Customers, error) {
	return f.customer, nil
}
func (f *fakeCustomerRepo) UpdateCustomer(ctx context.Context, id uuid.UUID, c *domain.Customers) error {
	return nil
}
func (f *fakeCustomerRepo) AddCustomer(ctx context.Context, c *domain.Customers) error { return nil }
func (f *fakeCustomerRepo) DeleteCustomer(ctx context.Context, id uuid.UUID) error     { return nil }
func (f *fakeCustomerRepo) GetAllCustomer(ctx context.Context, search string, limit, offset int) ([]*domain.Customers, error) {
	return nil, nil
}

type fakeDebtRepo struct{}

func (f *fakeDebtRepo) AddDebt(ctx context.Context, debt *domain.Debts) error           { return nil }
func (f *fakeDebtRepo) GetDebtByID(ctx context.Context, id uuid.UUID) (*domain.Debts, error) {
	return nil, nil
}
func (f *fakeDebtRepo) UpdateDebt(ctx context.Context, id uuid.UUID, debt *domain.Debts) error {
	return nil
}
func (f *fakeDebtRepo) DeleteDebt(ctx context.Context, id uuid.UUID) error { return nil }
func (f *fakeDebtRepo) GetAllDebt(ctx context.Context, filter domain.FilterDebt) (*domain.DebtsPaginated, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newProduct(price float64, stock float64) *domain.Products {
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
	return NewTransactionUsecase(trx, prod, &fakeUserRepo{user: user}, cust, debt, "Toko Ibu", zap.NewNop())
}

func customerFound() *[]domain.Customers {
	return &[]domain.Customers{{ID: uuid.New(), Name: "Budi"}}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// Penjualan tunai: total & subtotal dihitung server-side dari harga produk,
// nilai total/subtotal dari client diabaikan, dan flag isHutang = false.
func TestAddTransaction_Tunai_ComputesTotalServerSide(t *testing.T) {
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
	if trxRepo.lastIsHutang {
		t.Fatal("tunai should pass isHutang = false")
	}
}

// Error dari repository (mis. stok tidak cukup yang dicek atomik di DB) harus
// diteruskan oleh usecase.
func TestAddTransaction_RepoErrorIsPropagated(t *testing.T) {
	prod := newProduct(10000, 2)
	prodRepo := &fakeProductRepo{products: map[uuid.UUID]*domain.Products{prod.ID: prod}}
	trxRepo := &fakeTrxRepo{createErr: errors.New("insufficient stock")}
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
		t.Fatal("expected error propagated from repository")
	}
	if trxRepo.created != nil {
		t.Fatal("created must remain nil when repository fails")
	}
}

// Hutang: harga yang dipakai adalah SellingPriceDebt, flag isHutang = true,
// dan pelanggan tervalidasi sebelum menulis.
func TestAddTransaction_Hutang_UsesDebtPriceAndFlag(t *testing.T) {
	prod := newProduct(10000, 10) // SellingPriceDebt = 11000
	prodRepo := &fakeProductRepo{products: map[uuid.UUID]*domain.Products{prod.ID: prod}}
	trxRepo := &fakeTrxRepo{}
	custRepo := &fakeCustomerRepo{customer: customerFound()}
	uc := buildUsecase(trxRepo, prodRepo, custRepo, &fakeDebtRepo{})

	cid := uuid.New().String()
	err := uc.AddTransaction(context.Background(), &requestdto.AddTransactionRequest{
		NoInvoice:   "INV-3",
		TypePayment: "hutang",
		UserId:      uuid.New().String(),
		CustomerId:  &cid,
		Details: []requestdto.AddTransactionDetailRequest{
			{ProductId: prod.ID.String(), Qty: 2}, // 2 * 11000 = 22000
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !trxRepo.lastIsHutang {
		t.Fatal("hutang should pass isHutang = true")
	}
	if trxRepo.created.TotalTransaction != 22000 {
		t.Fatalf("total = %v, want 22000 (uses SellingPriceDebt)", trxRepo.created.TotalTransaction)
	}
	if d := trxRepo.created.TransactionDetail[0]; d.Price != 11000 {
		t.Fatalf("detail price = %v, want 11000", d.Price)
	}
}

// Nomor invoice duplikat -> ditolak sebelum membuat transaksi.
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
	if trxRepo.created != nil {
		t.Fatal("transaction must not be created on duplicate invoice")
	}
}

// Hutang tanpa customer_id -> ditolak.
func TestAddTransaction_Hutang_RequiresCustomer(t *testing.T) {
	prod := newProduct(10000, 10)
	prodRepo := &fakeProductRepo{products: map[uuid.UUID]*domain.Products{prod.ID: prod}}
	trxRepo := &fakeTrxRepo{}
	uc := buildUsecase(trxRepo, prodRepo, &fakeCustomerRepo{}, &fakeDebtRepo{})

	err := uc.AddTransaction(context.Background(), &requestdto.AddTransactionRequest{
		NoInvoice:   "INV-6",
		TypePayment: "hutang",
		UserId:      uuid.New().String(),
		Details:     []requestdto.AddTransactionDetailRequest{{ProductId: prod.ID.String(), Qty: 1}},
	})
	if err == nil {
		t.Fatal("expected error when hutang has no customer_id")
	}
	if trxRepo.created != nil {
		t.Fatal("transaction must not be created without customer for hutang")
	}
}

// Hutang dengan customer yang tidak ditemukan -> ditolak.
func TestAddTransaction_Hutang_CustomerNotFound(t *testing.T) {
	prod := newProduct(10000, 10)
	prodRepo := &fakeProductRepo{products: map[uuid.UUID]*domain.Products{prod.ID: prod}}
	trxRepo := &fakeTrxRepo{}
	custRepo := &fakeCustomerRepo{customer: nil} // tidak ditemukan
	uc := buildUsecase(trxRepo, prodRepo, custRepo, &fakeDebtRepo{})

	cid := uuid.New().String()
	err := uc.AddTransaction(context.Background(), &requestdto.AddTransactionRequest{
		NoInvoice:   "INV-7",
		TypePayment: "hutang",
		UserId:      uuid.New().String(),
		CustomerId:  &cid,
		Details:     []requestdto.AddTransactionDetailRequest{{ProductId: prod.ID.String(), Qty: 1}},
	})
	if err == nil {
		t.Fatal("expected error when hutang customer not found")
	}
	if trxRepo.created != nil {
		t.Fatal("transaction must not be created when customer not found")
	}
}

// Qty pecahan diterima (mendukung satuan gram/kg) dan subtotal dihitung dari
// harga * qty.
func TestAddTransaction_FractionalQtyAccepted(t *testing.T) {
	prod := newProduct(10000, 10)
	prodRepo := &fakeProductRepo{products: map[uuid.UUID]*domain.Products{prod.ID: prod}}
	trxRepo := &fakeTrxRepo{}
	uc := buildUsecase(trxRepo, prodRepo, &fakeCustomerRepo{}, &fakeDebtRepo{})

	err := uc.AddTransaction(context.Background(), &requestdto.AddTransactionRequest{
		NoInvoice:   "INV-8",
		TypePayment: "tunai",
		UserId:      uuid.New().String(),
		Details:     []requestdto.AddTransactionDetailRequest{{ProductId: prod.ID.String(), Qty: 1.5}},
	})
	if err != nil {
		t.Fatalf("fractional qty should be accepted, got: %v", err)
	}
	if trxRepo.created == nil {
		t.Fatal("transaction was not created")
	}
	if trxRepo.created.TotalTransaction != 15000 {
		t.Fatalf("total = %v, want 15000 (1.5 * 10000)", trxRepo.created.TotalTransaction)
	}
}
