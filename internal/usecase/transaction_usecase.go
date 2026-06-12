package usecase

import (
	"context"
	"errors"
	"fmt"
	"shop_project_be/internal/constant/enum"
	"shop_project_be/internal/constant/paginated"
	"shop_project_be/internal/domain"
	requestdto "shop_project_be/internal/dto/request_dto"
	responsedto "shop_project_be/internal/dto/response_dto"
	"shop_project_be/pkg/pdf"
	"strconv"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type transactionUsecase struct {
	trxRepo      domain.TransactionRepository
	productRepo  domain.ProductRepository
	userRepo     domain.UserRepository
	customerRepo domain.CustomerRepository
	debtRepo     domain.DebtRepository
	txManager    domain.TxManager
	storeName    string
	log          *zap.Logger
}

func NewTransactionUsecase(trxRepo domain.TransactionRepository, productRepo domain.ProductRepository, userRepo domain.UserRepository, customerRepo domain.CustomerRepository, debtRepo domain.DebtRepository, txManager domain.TxManager, storeName string, log *zap.Logger) domain.TransactionUsecase {
	return &transactionUsecase{
		trxRepo:      trxRepo,
		productRepo:  productRepo,
		userRepo:     userRepo,
		customerRepo: customerRepo,
		debtRepo:     debtRepo,
		txManager:    txManager,
		storeName:    storeName,
		log:          log,
	}
}

// AddTransaction implements [domain.TransactionUsecase].
//
// Membuat transaksi penjualan baru. Bagian VALIDASI (invoice unik, user ada,
// tipe bayar valid, customer ada untuk hutang) dilakukan di luar transaksi.
// Bagian PENULISAN yang harus konsisten (kurangi stok, hitung total dari harga
// produk, catat/akumulasi hutang, insert transaksi) dijalankan di dalam
// txManager.Do sehingga semuanya ATOMIK: bila ada satu langkah gagal, seluruh
// perubahan di-rollback. Repo (productRepo, debtRepo, dst) dipakai apa adanya —
// mereka otomatis ikut transaksi yang sama lewat dbtx.Conn.
func (t *transactionUsecase) AddTransaction(ctx context.Context, dto *requestdto.AddTransactionRequest) error {
	// 1. Pastikan nomor invoice belum pernah dipakai.
	existing, err := t.trxRepo.CheckTransactionByNoInvoice(ctx, dto.NoInvoice)
	if err != nil {
		t.log.Error("failed to check transaction", zap.Error(err))
		return fmt.Errorf("failed to check transaction")
	}
	if existing != nil {
		t.log.Warn("duplicate invoice", zap.String("no_invoice", dto.NoInvoice))
		return fmt.Errorf("transaction with no invoice %s already exists", dto.NoInvoice)
	}

	// 2. user_id berasal dari token (di-set handler). Parse aman tanpa panic.
	userID, err := uuid.Parse(dto.UserId)
	if err != nil {
		t.log.Error("invalid user id", zap.Error(err))
		return fmt.Errorf("invalid user id format")
	}
	user, err := t.userRepo.GetUserById(ctx, userID)
	if err != nil {
		t.log.Error("failed to get user", zap.Error(err))
		return fmt.Errorf("failed to get user")
	}
	if user == nil {
		t.log.Error("user not found", zap.String("user_id", dto.UserId))
		return fmt.Errorf("user not found")
	}

	// 3. Tipe pembayaran (tunai/hutang/transfer/qris).
	paymentType, err := enum.ParseMoneyPayment(dto.TypePayment)
	if err != nil {
		t.log.Error("failed to parse payment type", zap.Error(err))
		return fmt.Errorf("invalid payment type")
	}

	// 4. customer_id opsional, tetapi wajib & harus valid untuk pembayaran hutang.
	var customerID *uuid.UUID
	if dto.CustomerId != nil && *dto.CustomerId != "" {
		parsed, err := uuid.Parse(*dto.CustomerId)
		if err != nil {
			t.log.Error("failed to parse customer id", zap.Error(err))
			return fmt.Errorf("invalid customer id format")
		}
		customerID = &parsed
	}
	if paymentType.String() == "hutang" {
		if customerID == nil {
			return fmt.Errorf("customer id is required for hutang")
		}
		customer, err := t.customerRepo.GetCustomer(ctx, *customerID)
		if err != nil {
			t.log.Error("failed to get customer", zap.Error(err))
			return fmt.Errorf("failed to get customer")
		}
		if customer == nil {
			return fmt.Errorf("customer not found")
		}
	}

	// 5. Bangun detail. Hanya ProductID & Qty yang dipakai dari client; harga &
	//    subtotal diisi server di dalam transaksi (anti price-tampering).
	if len(dto.Details) == 0 {
		return fmt.Errorf("transaction details are required")
	}
	details := make([]domain.TransactionsDetail, 0, len(dto.Details))
	for _, d := range dto.Details {
		productID, err := uuid.Parse(d.ProductId)
		if err != nil {
			t.log.Error("failed to parse product id", zap.Error(err))
			return fmt.Errorf("invalid product id format")
		}
		details = append(details, domain.TransactionsDetail{
			ProductID: productID,
			Qty:       d.Qty,
		})
	}

	data := &domain.Transactions{
		NoInvoice:         dto.NoInvoice,
		UserID:            userID,
		CustomerID:        customerID,
		PaymentType:       paymentType,
		TransactionDetail: details,
	}

	// 6. Tulis semuanya dalam satu transaksi database (atomik).
	return t.txManager.Do(ctx, func(ctx context.Context) error {
		var total float64

		for i := range data.TransactionDetail {
			detail := &data.TransactionDetail[i]

			// Stok berupa bilangan bulat -> validasi qty dulu (tolak lebih awal).
			qty := int(detail.Qty)
			if float64(qty) != detail.Qty {
				return fmt.Errorf("qty produk %s harus bilangan bulat", detail.ProductID)
			}

			// Kunci & kurangi stok dulu (menolak bila stok tidak cukup / produk
			// tidak ada). Setelah baris terkunci, harga tidak bisa berubah oleh
			// transaksi lain saat kita membacanya.
			if err := t.productRepo.UpdateStockWithLock(ctx, detail.ProductID, -qty); err != nil {
				t.log.Error("failed to reduce stock", zap.Error(err))
				return err
			}

			// Baca harga dari produk yang sudah terkunci (anti price-tampering &
			// anti perubahan harga di tengah transaksi).
			product, err := t.productRepo.GetProduct(ctx, detail.ProductID)
			if err != nil {
				t.log.Error("failed to get product", zap.Error(err))
				return fmt.Errorf("product %s not found", detail.ProductID)
			}
			detail.Price = product.SellingPrice
			detail.PriceDebt = product.SellingPriceDebt
			detail.Subtotal = product.SellingPrice * detail.Qty
			total += detail.Subtotal
		}

		// Total final dihitung server-side, nilai dari client diabaikan.
		data.TotalTransaction = total

		// Penjualan hutang: cek apakah customer sudah punya hutang.
		//   - belum -> buat hutang baru (TotalDebt == RemainingDebt).
		//   - sudah -> akumulasikan (TotalDebt & RemainingDebt sama-sama bertambah).
		if paymentType.String() == "hutang" {
			// Kunci customer dulu agar dua transaksi hutang bersamaan untuk
			// customer yang sama tidak balapan membuat hutang ganda.
			if err := t.customerRepo.LockCustomerForUpdate(ctx, *customerID); err != nil {
				t.log.Error("failed to lock customer", zap.Error(err))
				return fmt.Errorf("failed to process debt")
			}

			debtID, err := t.customerRepo.GetDebtIdByCustomerId(ctx, *customerID)
			if err != nil {
				t.log.Error("failed to get debt id", zap.Error(err))
				return fmt.Errorf("failed to get debt")
			}

			if debtID == nil {
				debt := &domain.Debts{
					CustomerID:    *customerID,
					TotalDebt:     total,
					RemainingDebt: total,
					Status:        enum.BELUM_LUNAS,
				}
				if err := t.debtRepo.AddDebt(ctx, debt); err != nil {
					t.log.Error("failed to create debt", zap.Error(err))
					return fmt.Errorf("failed to create debt")
				}
				data.DebtID = &debt.ID
			} else {
				debt, err := t.debtRepo.GetDebtByID(ctx, *debtID)
				if err != nil {
					t.log.Error("failed to get debt", zap.Error(err))
					return fmt.Errorf("failed to get debt")
				}
				debt.TotalDebt += total
				debt.RemainingDebt += total
				debt.Status = enum.BELUM_LUNAS
				if err := t.debtRepo.UpdateDebt(ctx, *debtID, debt); err != nil {
					t.log.Error("failed to update debt", zap.Error(err))
					return fmt.Errorf("failed to update debt")
				}
				data.DebtID = debtID
			}
		}

		// Insert transaksi + detailnya.
		if err := t.trxRepo.CreateTransaction(ctx, data); err != nil {
			// Invoice kembar (race lolos dari pengecekan awal) -> pesan ramah.
			if errors.Is(err, domain.ErrDuplicateInvoice) {
				return fmt.Errorf("transaction with no invoice %s already exists", data.NoInvoice)
			}
			t.log.Error("failed to create transaction", zap.Error(err))
			return fmt.Errorf("failed to create transaction")
		}
		return nil
	})
}

// DeleteTransaction implements [domain.TransactionUsecase].
func (t *transactionUsecase) DeleteTransaction(ctx context.Context, dto *requestdto.DeleteTransactionRequest) error {
	// Parse aman: id tidak valid -> 400, bukan panic (uuid.Must akan panic).
	trxID, err := uuid.Parse(dto.ID)
	if err != nil {
		t.log.Error("invalid transaction id", zap.Error(err))
		return fmt.Errorf("invalid transaction id format")
	}

	if err := t.trxRepo.DeleteTransaction(ctx, trxID); err != nil {
		t.log.Error("transaction delete fail", zap.Error(err))
		return err
	}

	return nil
}

// GetAllTransaction implements [domain.TransactionUsecase].
func (t *transactionUsecase) GetAllTransaction(ctx context.Context, dto *requestdto.FilterTransactionRequest) ([]*responsedto.TransactionResponse, error) {
	var afterTimeDate time.Time
	var afterId uuid.UUID

	if dto.AfterTime != nil && dto.AfterID != nil {
		parsedId, err := uuid.Parse(*dto.AfterID)
		if err != nil {
			t.log.Error("failed to parse after_id", zap.Error(err))
			return nil, fmt.Errorf("invalid after_id format")
		}
		afterId = parsedId

		parsedTime, err := time.Parse(time.RFC3339, *dto.AfterTime)
		if err != nil {
			t.log.Error("failed to parse after_time", zap.Error(err))
			return nil, fmt.Errorf("invalid after_time format")
		}
		afterTimeDate = parsedTime
	}

	filter := &domain.FilterTransaction{
		NoInvoices: dto.InvoiceNumber,
		Cursor: &paginated.CursorMeta{
			AfterTime: afterTimeDate,
			AfterID:   afterId,
		},
		DateStart: dto.DateStart,
		DateEnd:   dto.DateEnd,
		Limit:     10,
		TypeTrx:   &dto.TypePayment,
	}

	result, err := t.trxRepo.GetAllTransaction(ctx, *filter)
	if err != nil {
		t.log.Error("failed to get all transactions", zap.Error(err))
		return nil, fmt.Errorf("failed to get all transactions")
	}

	if result == nil {
		return nil, nil
	}

	var responses []*responsedto.TransactionResponse
	for _, trx := range result.DataItem {
		var details []*responsedto.ProductTransactionResponse
		for _, d := range trx.TransactionDetail {
			details = append(details, &responsedto.ProductTransactionResponse{
				ProductName: d.Product.ProductName,
				Price:       d.Price,
				Qty:         d.Qty,
				Subtotal:    d.Subtotal,
			})
		}
		responses = append(responses, &responsedto.TransactionResponse{
			InvoiceNumber:      trx.NoInvoice,
			PaymentType:        int(trx.PaymentType),
			TotalTransaction:   trx.TotalTransaction,
			CreatedAt:          trx.CreatedAt.Format(time.RFC3339),
			TransactionDetails: details,
		})
	}

	return responses, nil
}

// GetTransaction implements [domain.TransactionUsecase].
func (t *transactionUsecase) GetTransaction(ctx context.Context, dto *requestdto.GetTransactionRequest) (*responsedto.TransactionResponse, error) {
	trxId, err := uuid.Parse(dto.ID)
	if err != nil {
		t.log.Error("failed to parse transaction id", zap.Error(err))
		return nil, fmt.Errorf("invalid transaction id format")
	}

	result, err := t.trxRepo.GetTransactionByID(ctx, trxId)
	if err != nil {
		t.log.Error("failed to get transaction", zap.Error(err))
		return nil, fmt.Errorf("failed to get transaction")
	}
	if result == nil {
		t.log.Error("transaction not found", zap.String("id", dto.ID))
		return nil, fmt.Errorf("transaction not found")
	}

	var transactionDetail []*responsedto.ProductTransactionResponse
	for _, d := range result.TransactionDetail {
		transactionDetail = append(transactionDetail, &responsedto.ProductTransactionResponse{
			ProductName: d.Product.ProductName,
			Price:       d.Price,
			Qty:         d.Qty,
			Subtotal:    d.Subtotal,
		})
	}

	response := &responsedto.TransactionResponse{
		InvoiceNumber:      result.NoInvoice,
		PaymentType:        int(result.PaymentType),
		TotalTransaction:   result.TotalTransaction,
		TotalProfit:        0,
		CreatedAt:          result.CreatedAt.Format(time.RFC3339),
		TransactionDetails: transactionDetail,
	}
	return response, nil
}

// PrintReportMonth implements [domain.TransactionUsecase].
// Membuat PDF laporan bulanan berisi total transaksi dan pendapatan selama
// satu bulan, lalu mengembalikan URL file yang bisa di-download client.
func (t *transactionUsecase) PrintReportMonth(ctx context.Context, dto *requestdto.PrintReportMonthRequest) (*responsedto.PrintReportMonthTransactionResponse, error) {
	month := dto.Month
	year := dto.Year
	now := time.Now()
	if month == 0 {
		month = int(now.Month())
	}
	if year == 0 {
		year = now.Year()
	}
	if month < 1 || month > 12 {
		t.log.Error("invalid month", zap.Int("month", month))
		return nil, fmt.Errorf("invalid month: %d", month)
	}

	userId, err := uuid.Parse(dto.UserId)
	if err != nil {
		t.log.Error("failed to parse user id", zap.Error(err))
		return nil, fmt.Errorf("invalid user id format")
	}
	user, err := t.userRepo.GetUserById(ctx, userId)
	if err != nil {
		t.log.Error("failed to get user", zap.Error(err))
		return nil, fmt.Errorf("failed to get user")
	}
	if user == nil {
		t.log.Error("user not found", zap.String("user_id", dto.UserId))
		return nil, fmt.Errorf("user not found")
	}

	report, err := t.trxRepo.GetMonthlyReport(ctx, month, year)
	if err != nil {
		t.log.Error("failed to get monthly report", zap.Error(err))
		return nil, fmt.Errorf("failed to get monthly report")
	}

	dailyReport, err := t.trxRepo.GetDailyReport(ctx, month, year)
	if err != nil {
		t.log.Error("failed to get daily report", zap.Error(err))
		return nil, fmt.Errorf("failed to get daily report")
	}

	daily := make([]pdf.MonthReportDailyRow, 0, len(dailyReport))
	for _, d := range dailyReport {
		daily = append(daily, pdf.MonthReportDailyRow{
			Date:             d.Date,
			TotalTransaction: d.TotalTransaction,
			Revenue:          d.TotalRevenue,
			Debt:             d.TotalDebt,
			Total:            d.GrandTotal,
		})
	}

	productSold, err := t.trxRepo.GetMonthlyProductSold(ctx, month, year)
	if err != nil {
		t.log.Error("failed to get monthly product sold", zap.Error(err))
		return nil, fmt.Errorf("failed to get monthly product sold")
	}

	products := make([]pdf.MonthReportProductRow, 0, len(productSold))
	for _, p := range productSold {
		products = append(products, pdf.MonthReportProductRow{
			ProductName: p.ProductName,
			Qty:         p.Qty,
			Total:       p.Total,
		})
	}

	dailyProductSold, err := t.trxRepo.GetDailyProductSold(ctx, month, year)
	if err != nil {
		t.log.Error("failed to get daily product sold", zap.Error(err))
		return nil, fmt.Errorf("failed to get daily product sold")
	}

	// Kelompokkan produk terjual per tanggal. Hasil repo sudah terurut menaik
	// per tanggal, jadi cukup pecah saat tanggalnya berganti.
	var dailyProducts []pdf.MonthReportDailyProducts
	for _, p := range dailyProductSold {
		n := len(dailyProducts)
		if n == 0 || !dailyProducts[n-1].Date.Equal(p.Date) {
			dailyProducts = append(dailyProducts, pdf.MonthReportDailyProducts{Date: p.Date})
			n++
		}
		dailyProducts[n-1].Products = append(dailyProducts[n-1].Products, pdf.MonthReportProductRow{
			ProductName: p.ProductName,
			Qty:         p.Qty,
			Total:       p.Total,
		})
	}

	urlPdf, err := pdf.GenerateMonthReport(pdf.MonthReportData{
		StoreName:        t.storeName,
		Cashier:          user.Username,
		Month:            month,
		Year:             year,
		TotalTransaction: report.TotalTransaction,
		TotalRevenue:     report.TotalRevenue,
		TotalDebt:        report.TotalDebt,
		GrandTotal:       report.GrandTotal,
		Daily:            daily,
		ProductsSold:     products,
		DailyProducts:    dailyProducts,
		GeneratedAt:      now,
	})
	if err != nil {
		t.log.Error("failed to generate month report pdf", zap.Error(err))
		return nil, fmt.Errorf("failed to generate report pdf")
	}

	return &responsedto.PrintReportMonthTransactionResponse{
		ID:     uuid.New(),
		Month:  fmt.Sprintf("%02d", month),
		Year:   strconv.Itoa(year),
		UrlPdf: urlPdf,
	}, nil
}

// PrintReportTransaction implements [domain.TransactionUsecase].
// Membuat PDF struk untuk satu transaksi (dicari via trx_id atau no_invoice)
// yang dapat diberikan ke customer, lalu mengembalikan URL file-nya.
func (t *transactionUsecase) PrintReportTransaction(ctx context.Context, dto *requestdto.PrintReportTransactionRequest) (*responsedto.PrintReportTransactionResponse, error) {
	var trxId uuid.UUID

	switch {
	case dto.TrxId != "":
		parsed, err := uuid.Parse(dto.TrxId)
		if err != nil {
			t.log.Error("failed to parse trx id", zap.Error(err))
			return nil, fmt.Errorf("invalid trx id format")
		}
		trxId = parsed
	case dto.NoInvoice != "":
		existing, err := t.trxRepo.CheckTransactionByNoInvoice(ctx, dto.NoInvoice)
		if err != nil {
			t.log.Error("failed to check transaction", zap.Error(err))
			return nil, fmt.Errorf("failed to get transaction")
		}
		if existing == nil {
			t.log.Error("transaction not found", zap.String("no_invoice", dto.NoInvoice))
			return nil, fmt.Errorf("transaction with no invoice %s not found", dto.NoInvoice)
		}
		trxId = existing.ID
	default:
		return nil, fmt.Errorf("trx_id or number_invoice is required")
	}

	// Ambil data lengkap (User, Customer, dan detail produk ter-preload).
	trx, err := t.trxRepo.GetTransactionByID(ctx, trxId)
	if err != nil {
		t.log.Error("failed to get transaction", zap.Error(err))
		return nil, fmt.Errorf("failed to get transaction")
	}
	if trx == nil {
		t.log.Error("transaction not found", zap.String("trx_id", trxId.String()))
		return nil, fmt.Errorf("transaction not found")
	}

	items := make([]pdf.TransactionReportItem, 0, len(trx.TransactionDetail))
	for _, d := range trx.TransactionDetail {
		items = append(items, pdf.TransactionReportItem{
			ProductName: d.Product.ProductName,
			Qty:         d.Qty,
			Price:       d.Price,
			Subtotal:    d.Subtotal,
		})
	}

	var customerName string
	if trx.CustomerID != nil {
		customerName = trx.Customer.Name
	}
	urlPdf, err := pdf.GenerateTransactionReport(pdf.TransactionReportData{
		StoreName:   t.storeName,
		NoInvoice:   trx.NoInvoice,
		Cashier:     trx.User.Username,
		Customer:    customerName,
		PaymentType: trx.PaymentType.String(),
		CreatedAt:   trx.CreatedAt,
		Items:       items,
		Total:       trx.TotalTransaction,
		GeneratedAt: time.Now(),
	})
	if err != nil {
		t.log.Error("failed to generate transaction report pdf", zap.Error(err))
		return nil, fmt.Errorf("failed to generate report pdf")
	}

	return &responsedto.PrintReportTransactionResponse{
		ID:        trx.ID,
		NoInvoice: trx.NoInvoice,
		UrlPdf:    urlPdf,
	}, nil
}
