package usecase

import (
	"context"
	"fmt"
	"shop_project_be/internal/constant/enum"
	"shop_project_be/internal/domain"
	requestdto "shop_project_be/internal/dto/request_dto"
	responsedto "shop_project_be/internal/dto/response_dto"
	"shop_project_be/pkg/pdf"
	"strconv"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// dateLayout adalah format tanggal yang dipakai pada input/output hutang.
const dateLayout = "2006-01-02"

type debtUsecase struct {
	debtRepo domain.DebtRepository
	log      *zap.Logger
}

func NewDebtUsecase(debtRepo domain.DebtRepository, log *zap.Logger) domain.DebtUseCase {
	return &debtUsecase{
		debtRepo: debtRepo,
		log:      log,
	}
}

// money memformat angka menjadi string dengan 2 desimal.
func money(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}

// toDebtResponse memetakan entitas Debts ke DTO response.
func toDebtResponse(debt *domain.Debts) responsedto.DebtResponseDto {
	var dateDebt *string
	if !debt.DueDate.IsZero() {
		s := debt.DueDate.Format(dateLayout)
		dateDebt = &s
	}

	transactions := make([]responsedto.TransactionResponse, 0, len(debt.Transactions))
	for _, trx := range debt.Transactions {
		transactions = append(transactions, responsedto.TransactionResponse{
			InvoiceNumber:    trx.NoInvoice,
			PaymentType:      int(trx.PaymentType),
			TotalTransaction: trx.TotalTransaction,
			CreatedAt:        trx.CreatedAt.Format(time.RFC3339),
		})
	}

	return responsedto.DebtResponseDto{
		NameCustomer:    debt.Customer.Name,
		TotalDebt:       money(debt.TotalDebt),
		RemainingDebt:   money(debt.RemainingDebt),
		DateDebt:        dateDebt,
		TransactionList: transactions,
	}
}

// AddingDebtCustomer implements [domain.DebtUseCase].
func (d *debtUsecase) AddingDebtCustomer(ctx context.Context, request *requestdto.AddDebtRequest) error {
	customerId, err := uuid.Parse(request.CustomerID)
	if err != nil {
		d.log.Error("failed to parse customer id", zap.Error(err))
		return fmt.Errorf("invalid customer id format")
	}

	dueDate, err := time.Parse(dateLayout, request.JatuhTempo)
	if err != nil {
		d.log.Error("failed to parse jatuh_tempo", zap.Error(err))
		return fmt.Errorf("invalid jatuh_tempo format (expected YYYY-MM-DD)")
	}

	debt := &domain.Debts{
		CustomerID:    customerId,
		TotalDebt:     request.TotalTransaksi,
		RemainingDebt: request.TotalTransaksi,
		Status:        enum.BELUM_LUNAS,
		DueDate:       dueDate,
	}
	if err := d.debtRepo.AddDebt(ctx, debt); err != nil {
		d.log.Error("failed to add debt", zap.Error(err))
		return fmt.Errorf("failed to add debt")
	}
	return nil
}

// DeleteDebtCustomer implements [domain.DebtUseCase].
func (d *debtUsecase) DeleteDebtCustomer(ctx context.Context, request *requestdto.DeleteDebtRequest) error {
	id, err := uuid.Parse(request.DebtId)
	if err != nil {
		d.log.Error("failed to parse debt id", zap.Error(err))
		return fmt.Errorf("invalid debt id format")
	}
	if err := d.debtRepo.DeleteDebt(ctx, id); err != nil {
		d.log.Error("failed to delete debt", zap.Error(err))
		return fmt.Errorf("failed to delete debt")
	}
	return nil
}

// GetAllDebtCustomerList implements [domain.DebtUseCase].
func (d *debtUsecase) GetAllDebtCustomerList(ctx context.Context, request *requestdto.FilterDebtRequest) (*[]responsedto.DebtResponseDto, error) {
	filter := domain.FilterDebt{Limit: 10}
	if request.CustomerId != "" {
		customerId, err := uuid.Parse(request.CustomerId)
		if err != nil {
			d.log.Error("failed to parse customer id", zap.Error(err))
			return nil, fmt.Errorf("invalid customer id format")
		}
		filter.CustomerID = customerId
	}

	result, err := d.debtRepo.GetAllDebt(ctx, filter)
	if err != nil {
		d.log.Error("failed to get debts", zap.Error(err))
		return nil, fmt.Errorf("failed to get debts")
	}

	responses := make([]responsedto.DebtResponseDto, 0, len(result.Data))
	for _, debt := range result.Data {
		responses = append(responses, toDebtResponse(debt))
	}
	return &responses, nil
}

// GetDebtCustomer implements [domain.DebtUseCase].
func (d *debtUsecase) GetDebtCustomer(ctx context.Context, request *requestdto.GetDebtRequest) (*responsedto.DebtResponseDto, error) {
	id, err := uuid.Parse(request.DebtId)
	if err != nil {
		d.log.Error("failed to parse debt id", zap.Error(err))
		return nil, fmt.Errorf("invalid debt id format")
	}

	debt, err := d.debtRepo.GetDebtByID(ctx, id)
	if err != nil {
		d.log.Error("failed to get debt", zap.Error(err))
		return nil, fmt.Errorf("failed to get debt")
	}
	if debt == nil {
		d.log.Error("debt not found", zap.String("debt_id", request.DebtId))
		return nil, fmt.Errorf("debt not found")
	}

	response := toDebtResponse(debt)
	return &response, nil
}

// PrintReportDebtCustomer implements [domain.DebtUseCase].
// Membuat PDF laporan hutang seorang customer (ringkasan + riwayat pembayaran)
// lalu mengembalikan URL file yang bisa di-download client.
func (d *debtUsecase) PrintReportDebtCustomer(ctx context.Context, request *requestdto.PrintDebtReport) (*responsedto.PrintDebtCustomerResponse, error) {
	if request.DebtId == "" {
		return nil, fmt.Errorf("debt_id is required")
	}
	id, err := uuid.Parse(request.DebtId)
	if err != nil {
		d.log.Error("failed to parse debt id", zap.Error(err))
		return nil, fmt.Errorf("invalid debt id format")
	}

	debt, err := d.debtRepo.GetDebtByID(ctx, id)
	if err != nil {
		d.log.Error("failed to get debt", zap.Error(err))
		return nil, fmt.Errorf("failed to get debt")
	}
	if debt == nil {
		d.log.Error("debt not found", zap.String("debt_id", request.DebtId))
		return nil, fmt.Errorf("debt not found")
	}

	payments := make([]pdf.DebtPaymentRow, 0, len(debt.DebtPayments))
	for _, p := range debt.DebtPayments {
		var cashier string
		if p.User != nil {
			cashier = p.User.Username
		}
		payments = append(payments, pdf.DebtPaymentRow{
			Date:    p.TanggalBayar,
			Cashier: cashier,
			Nominal: p.NominalBayar,
		})
	}

	urlPdf, err := pdf.GenerateDebtReport(pdf.DebtReportData{
		DebtID:          debt.ID.String(),
		CustomerName:    debt.Customer.Name,
		CustomerPhone:   debt.Customer.Phone,
		CustomerAddress: debt.Customer.Address,
		TotalDebt:       debt.TotalDebt,
		RemainingDebt:   debt.RemainingDebt,
		Status:          debt.Status.String(),
		DueDate:         debt.DueDate,
		Payments:        payments,
		GeneratedAt:     time.Now(),
	})
	if err != nil {
		d.log.Error("failed to generate debt report pdf", zap.Error(err))
		return nil, fmt.Errorf("failed to generate report pdf")
	}

	return &responsedto.PrintDebtCustomerResponse{
		CustomerName: debt.Customer.Name,
		DebtId:       debt.ID.String(),
		UrlPdf:       urlPdf,
	}, nil
}
