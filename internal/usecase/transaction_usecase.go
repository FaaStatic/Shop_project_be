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
	"strings"
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

// invoicePrefix is the mandatory prefix for every transaction no_invoice, shared
// by cash transactions (AddTransaction) and payments (generateInvoice).
const invoicePrefix = "INV-"

// ensureInvoicePrefix returns a no_invoice guaranteed to start with "INV-".
// Idempotent and case-insensitive so an already-correct invoice is not
// duplicated. If empty, a new invoice is generated.
func ensureInvoicePrefix(noInvoice string) string {
	trimmed := strings.TrimSpace(noInvoice)
	if trimmed == "" {
		return generateInvoice()
	}
	if strings.HasPrefix(strings.ToUpper(trimmed), invoicePrefix) {
		return trimmed
	}
	return invoicePrefix + trimmed
}

// AddTransaction implements [domain.TransactionUsecase].
func (t *transactionUsecase) AddTransaction(ctx context.Context, dto *requestdto.AddTransactionRequest) error {
	return t.addTransaction(ctx, dto, true)
}

// AddPrepaidTransaction implements [domain.TransactionUsecase]: a transaction from
// an online payment whose stock was already reserved at charge time, so stock
// is not deducted again here.
func (t *transactionUsecase) AddPrepaidTransaction(ctx context.Context, dto *requestdto.AddTransactionRequest) error {
	return t.addTransaction(ctx, dto, false)
}

func (t *transactionUsecase) addTransaction(ctx context.Context, dto *requestdto.AddTransactionRequest, deductStock bool) error {
	// Match the payment format: no_invoice always starts with "INV-".
	// Idempotent — an invoice from a payment (order_id) already prefixed with INV-
	// is not duplicated into "INV-INV-...".
	noInvoice := ensureInvoicePrefix(dto.NoInvoice)

	check, err := t.trxRepo.CheckTransactionByNoInvoice(ctx, noInvoice)
	if err != nil {
		t.log.Error("failed to check transaction", zap.Error(err))
		return fmt.Errorf("failed to check transaction")
	}
	if check != nil {
		t.log.Error("transaction with no invoice %s already exists", zap.String("no_invoice", noInvoice))
		return fmt.Errorf("transaction with no invoice %s already exists", noInvoice)
	}

	userId, err := uuid.Parse(dto.UserId)
	if err != nil {
		t.log.Error("failed to parse user id", zap.Error(err))
		return fmt.Errorf("invalid user id format")
	}

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

	paymentType, err := enum.ParseMoneyPayment(dto.TypePayment)
	if err != nil {
		t.log.Error("failed to parse payment type", zap.Error(err))
		return fmt.Errorf("failed to parse payment type")
	}
	isHutang := paymentType.String() == "hutang"

	isTransfer := paymentType.String() == "transfer"
	var bank *string
	if isTransfer {
		if dto.Bank == nil || (*dto.Bank != "bca" && *dto.Bank != "mandiri") {
			t.log.Error("bank is required for transfer", zap.String("no_invoice", noInvoice))
			return fmt.Errorf("bank (bca/mandiri) is required for transfer payment")
		}
		bank = dto.Bank
	}

	// Compute subtotal & total on the server; do not trust values from the client.
	// The debt price (SellingPriceDebt) is used when the payment is a debt.
	var detailTrx []domain.TransactionsDetail
	var total float64
	for _, detail := range dto.Details {
		productId, err := uuid.Parse(detail.ProductId)
		if err != nil {
			t.log.Error("failed to parse product id", zap.Error(err))
			return fmt.Errorf("invalid product id format")
		}
		product, err := t.productRepo.GetProduct(ctx, productId)
		if err != nil {
			t.log.Error("failed to get product", zap.Error(err))
			return fmt.Errorf("failed to get product")
		}
		if product == nil {
			t.log.Error("product not found", zap.String("product_id", detail.ProductId))
			return fmt.Errorf("product not found")
		}

		unitPrice := product.SellingPrice
		if isHutang && !product.ProductType.IsDigital() {
			unitPrice = product.SellingPriceDebt
		}

		var destination *string
		if product.ProductType.IsDigital() {
			if detail.Destination == nil || strings.TrimSpace(*detail.Destination) == "" {
				t.log.Error("destination required for digital product", zap.String("product_id", detail.ProductId))
				return fmt.Errorf("destination is required for digital product %s", detail.ProductId)
			}
			d := strings.TrimSpace(*detail.Destination)
			destination = &d
		}

		subtotal := unitPrice * detail.Qty
		total += subtotal

		detailTrx = append(detailTrx, domain.TransactionsDetail{
			ProductID:   productId,
			Price:       unitPrice,
			PriceDebt:   product.SellingPriceDebt,
			Qty:         detail.Qty,
			Subtotal:    subtotal,
			Destination: destination,
		})
	}

	// A debt must have a valid customer (validated before writing).
	if isHutang {
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
	}

	data := &domain.Transactions{
		NoInvoice:         noInvoice,
		UserID:            userId,
		CustomerID:        customerId,
		PaymentType:       paymentType,
		Bank:              bank,
		TotalTransaction:  total,
		TransactionDetail: detailTrx,
	}

	// Stock is decremented, the debt is upserted, and the transaction is saved in one
	// DB transaction: all succeed or are rolled back together.
	if err := t.trxRepo.CreateTransaction(ctx, data, isHutang, deductStock); err != nil {
		t.log.Error("failed to create transaction", zap.Error(err))
		return err
	}

	return nil
}

// DeleteTransaction implements [domain.TransactionUsecase].
func (t *transactionUsecase) DeleteTransaction(ctx context.Context, dto *requestdto.DeleteTransactionRequest) error {
	trxId, err := uuid.Parse(dto.ID)
	if err != nil {
		t.log.Error("transaction id parse fail", zap.Error(err))
		return fmt.Errorf("invalid transaction id format")
	}
	// Restoring stock, reversing the debt balance, and deleting the transaction are done
	// atomically in the repository (all in one DB transaction).
	if err := t.trxRepo.DeleteTransaction(ctx, trxId); err != nil {
		t.log.Error("transaction delete fail", zap.Error(err))
		return err
	}

	return nil
}

// GetAllTransaction implements [domain.TransactionUsecase].
func (t *transactionUsecase) GetAllTransaction(ctx context.Context, dto *requestdto.FilterTransactionRequest) (*responsedto.GetAllTransactionResponse, error) {
	// Cursor pagination is optional. On the first page after_id/after_time are not yet
	// present, so empty/absent parameters are treated as "no cursor".
	// The cursor is left nil so the repository does not apply a created_at filter
	// with a zero-time (which would empty the first page).
	var afterId, afterTimeRaw string
	if dto.AfterID != nil {
		afterId = strings.TrimSpace(*dto.AfterID)
	}
	if dto.AfterTime != nil {
		afterTimeRaw = strings.TrimSpace(*dto.AfterTime)
	}

	var cursor *paginated.CursorMeta
	if afterId != "" && afterTimeRaw != "" {
		parsedId, err := uuid.Parse(afterId)
		if err != nil {
			t.log.Error("failed to parse after_id", zap.Error(err))
			return nil, fmt.Errorf("invalid after_id format")
		}
		parsedTime, err := time.Parse(paginated.TimeLayout, afterTimeRaw)
		if err != nil {
			t.log.Error("failed to parse after_time", zap.Error(err))
			return nil, fmt.Errorf("invalid after_time format")
		}
		cursor = &paginated.CursorMeta{
			AfterTime: parsedTime,
			AfterID:   parsedId,
		}
	}

	filter := &domain.FilterTransaction{
		NoInvoices: dto.InvoiceNumber,
		Cursor:     cursor,
		DateStart:  dto.DateStart,
		DateEnd:    dto.DateEnd,
		Limit:      10,
		TypeTrx:    &dto.TypePayment,
	}

	result, err := t.trxRepo.GetAllTransaction(ctx, *filter)
	if err != nil {
		t.log.Error("failed to get all transactions", zap.Error(err))
		return nil, fmt.Errorf("failed to get all transactions")
	}

	if result == nil {
		return nil, nil
	}

	responses := make([]*responsedto.TransactionResponse, 0, len(result.DataItem))
	for _, trx := range result.DataItem {
		details := make([]*responsedto.ProductTransactionResponse, 0, len(trx.TransactionDetail))
		for _, d := range trx.TransactionDetail {
			details = append(details, &responsedto.ProductTransactionResponse{
				ProductID:   d.ProductID,
				ProductName: d.Product.ProductName,
				Price:       d.Price,
				Qty:         d.Qty,
				Subtotal:    d.Subtotal,
			})
		}
		responses = append(responses, &responsedto.TransactionResponse{
			TransactionID:      trx.ID,
			InvoiceNumber:      trx.NoInvoice,
			PaymentType:        int(trx.PaymentType),
			TotalTransaction:   trx.TotalTransaction,
			CreatedAt:          trx.CreatedAt.Format(time.RFC3339),
			TransactionDetails: details,
		})
	}

	// Cursor is nil on the last page; Encode is nil-safe (no panic).
	nextId, nextTime := result.Cursor.Encode()

	return &responsedto.GetAllTransactionResponse{
		UserID:          dto.UserId,
		AfterId:         nextId,
		AfterTime:       nextTime,
		HasNext:         result.HasNext,
		TransactionList: responses,
	}, nil
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
// Builds a monthly PDF report with total transactions and revenue over
// one month, then returns the file URL the client can download.
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

	// Group products sold per date. The repo result is already sorted ascending
	// by date, so just split when the date changes.
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
// Builds a PDF receipt for a single transaction (looked up via trx_id or no_invoice)
// that can be given to the customer, then returns its file URL.
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

	// Fetch the full data (User, Customer, and product details preloaded).
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
