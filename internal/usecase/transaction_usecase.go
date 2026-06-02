package usecase

import (
	"context"
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
	storeName    string
	log          *zap.Logger
}

func NewTransactionUsecase(trxRepo domain.TransactionRepository, productRepo domain.ProductRepository, userRepo domain.UserRepository, customerRepo domain.CustomerRepository, debtRepo domain.DebtRepository, storeName string, log *zap.Logger) domain.TransactionUsecase {
	return &transactionUsecase{
		trxRepo:      trxRepo,
		productRepo:  productRepo,
		userRepo:     userRepo,
		customerRepo: customerRepo,
		debtRepo:     debtRepo,
		storeName:    storeName,
		log:          log,
	}
}

// AddTransaction implements [domain.TransactionUsecase].
func (t *transactionUsecase) AddTransaction(ctx context.Context, dto *requestdto.AddTransactionRequest) error {
	check, err := t.trxRepo.CheckTransactionByNoInvoice(ctx, dto.NoInvoice)
	if err != nil {
		t.log.Error("failed to check transaction", zap.Error(err))
		return fmt.Errorf("failed to check transaction")
	}
	if check != nil {
		t.log.Error("transaction with no invoice %s already exists", zap.String("no_invoice", dto.NoInvoice))
		return fmt.Errorf("transaction with no invoice %s already exists", dto.NoInvoice)
	}

	userId := uuid.Must(uuid.Parse(dto.UserId))
	user, err := t.userRepo.GetUserById(ctx, userId)
	if err != nil {
		t.log.Error("failed to get user", zap.Error(err))
		return fmt.Errorf("failed to get user")
	}
	if user == nil {
		t.log.Error("user not found", zap.String("user_id", dto.UserId))
		return fmt.Errorf("user not found")
	}

	var customerId *uuid.UUID
	if dto.CustomerId != nil {
		parsedID, err := uuid.Parse(*dto.CustomerId)
		if err != nil {
			t.log.Error("failed to parse customer ID", zap.Error(err))
			return fmt.Errorf("failed to parse customer ID")
		}
		customerId = &parsedID
	}

	var detailTrx []domain.TransactionsDetail

	for _, detail := range dto.Details {
		productId := uuid.Must(uuid.Parse(detail.ProductId))
		product, err := t.productRepo.GetProduct(ctx, productId)
		if err != nil {
			t.log.Error("failed to get product", zap.Error(err))
			return fmt.Errorf("failed to get product")
		}
		if product == nil {
			t.log.Error("product not found", zap.String("product_id", detail.ProductId))
			return fmt.Errorf("product not found")
		}

		detailTrx = append(detailTrx, domain.TransactionsDetail{
			ProductID: productId,
			Price:     product.SellingPrice,
			PriceDebt: product.SellingPriceDebt,
			Qty:       detail.Qty,
			Subtotal:  detail.Subtotal,
		})

	}

	paymentType, err := enum.ParseMoneyPayment(dto.TypePayment)
	if err != nil {
		t.log.Error("failed to parse payment type", zap.Error(err))
		return fmt.Errorf("failed to parse payment type")
	}

	data := &domain.Transactions{
		NoInvoice:         dto.NoInvoice,
		UserID:            uuid.Must(uuid.Parse(dto.UserId)),
		CustomerID:        customerId,
		DebtID:            nil,
		PaymentType:       paymentType,
		TotalTransaction:  dto.TotalTransaction,
		TransactionDetail: detailTrx,
	}

	if paymentType.String() == "hutang" {
		if customerId == nil {
			t.log.Error("customer id is required for hutang")
			return fmt.Errorf("customer id is required for hutang")
		}
		customer, err := t.customerRepo.GetCustomer(ctx, *customerId)
		if err != nil {
			t.log.Error("failed to get customer", zap.Error(err))
			return fmt.Errorf("failed to get customer")
		}
		if customer == nil {
			t.log.Error("customer not found", zap.String("customer_id", *dto.CustomerId))
			return fmt.Errorf("customer not found")
		}

		debtId, err := t.customerRepo.GetDebtIdByCustomerId(ctx, *customerId)
		if err != nil {
			t.log.Error("failed to get debt id by customer id", zap.Error(err))
			return fmt.Errorf("failed to get debt id by customer id")
		}
		if debtId == nil {
			debt := &domain.Debts{
				CustomerID: *customerId,
				TotalDebt:  data.TotalTransaction,
				Status:     enum.BELUM_LUNAS,
			}
			err = t.debtRepo.AddDebt(ctx, debt)
			if err != nil {
				t.log.Error("failed to create debt", zap.Error(err))
				return fmt.Errorf("failed to create debt")
			}
		}
		if debtId != nil {
			debt, err := t.debtRepo.GetDebtByID(ctx, *debtId)
			if err != nil {
				t.log.Error("failed to get debt", zap.Error(err))
				return fmt.Errorf("failed to get debt")
			}
			if debt == nil {
				t.log.Error("debt not found", zap.String("debt_id", debtId.String()))
				return fmt.Errorf("debt not found")
			}
			debt.TotalDebt += data.TotalTransaction
			err = t.debtRepo.UpdateDebt(ctx, *debtId, debt)
			if err != nil {
				t.log.Error("failed to update debt", zap.Error(err))
				return fmt.Errorf("failed to update debt")
			}
		}

	}

	result := t.trxRepo.CreateTransaction(ctx, data)
	if result != nil {
		return fmt.Errorf("failed to create transaction: %w", result)
	}

	return nil

}

// DeleteTransaction implements [domain.TransactionUsecase].
func (t *transactionUsecase) DeleteTransaction(ctx context.Context, dto *requestdto.DeleteTransactionRequest) error {
	productId := uuid.Must(uuid.Parse(dto.ID))
	error := t.trxRepo.DeleteTransaction(ctx, productId)

	if error != nil {
		t.log.Error("transaction delete fail", zap.Error(error))
		return error
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
