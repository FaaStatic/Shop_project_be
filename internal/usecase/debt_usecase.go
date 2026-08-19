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
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// dateLayout is the date format used for debt input/output.
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

// toDebtResponse maps a Debts entity to the response DTO.
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
		TotalDebt:       debt.TotalDebt,
		RemainingDebt:   debt.RemainingDebt,
		DateDebt:        dateDebt,
		TransactionList: transactions,
	}
}

// AddingDebtCustomer implements [domain.DebtUseCase].
func (d *debtUsecase) AddingDebtCustomer(ctx context.Context, request *requestdto.AddDebtRequest) error {
	customerId, err := uuid.Parse(request.CustomerID)
	if err != nil {
		d.log.Error("failed to parse customer id", zap.Error(err))
		return domain.InvalidID("invalid customer id format")
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
		return fmt.Errorf("failed to add debt: %w", domain.ErrInternal)
	}
	return nil
}

// DeleteDebtCustomer implements [domain.DebtUseCase].
func (d *debtUsecase) DeleteDebtCustomer(ctx context.Context, request *requestdto.DeleteDebtRequest) error {
	id, err := uuid.Parse(request.DebtId)
	if err != nil {
		d.log.Error("failed to parse debt id", zap.Error(err))
		return domain.InvalidID("invalid debt id format")
	}
	if err := d.debtRepo.DeleteDebt(ctx, id); err != nil {
		d.log.Error("failed to delete debt", zap.Error(err))
		if errors.Is(err, domain.ErrNotFound) {
			return err
		}
		return fmt.Errorf("failed to delete debt: %w", domain.ErrInternal)
	}
	return nil
}

// GetAllDebtCustomerList implements [domain.DebtUseCase].
func (d *debtUsecase) GetAllDebtCustomerList(ctx context.Context, request *requestdto.FilterDebtRequest) (*responsedto.DebtListResponseDto, error) {
	filter := domain.FilterDebt{Limit: request.Limit, Order: request.Order}
	if request.CustomerId != "" {
		customerId, err := uuid.Parse(request.CustomerId)
		if err != nil {
			d.log.Error("failed to parse customer id", zap.Error(err))
			return nil, domain.InvalidID("invalid customer id format")
		}
		filter.CustomerID = customerId
	}

	// Cursor is optional. The first page has no after_id/after_time yet, so
	// both must be set for the cursor to apply; otherwise leave it nil so the
	// repo does not filter created_at with a zero-time (which empties the result).
	var afterId, afterTimeRaw string
	if request.AfterID != nil {
		afterId = strings.TrimSpace(*request.AfterID)
	}
	if request.AfterTime != nil {
		afterTimeRaw = strings.TrimSpace(*request.AfterTime)
	}
	if afterId != "" && afterTimeRaw != "" {
		afterTime, err := time.Parse(paginated.TimeLayout, afterTimeRaw)
		if err != nil {
			d.log.Error("failed to parse after_time", zap.Error(err))
			return nil, fmt.Errorf("invalid after_time format")
		}
		afterUUID, err := uuid.Parse(afterId)
		if err != nil {
			d.log.Error("failed to parse after_id", zap.Error(err))
			return nil, domain.InvalidID("invalid after_id format")
		}
		filter.Cursor = &paginated.CursorMeta{AfterTime: afterTime, AfterID: afterUUID}
	}

	result, err := d.debtRepo.GetAllDebt(ctx, filter)
	if err != nil {
		d.log.Error("failed to get debts", zap.Error(err))
		return nil, fmt.Errorf("failed to get debts: %w", domain.ErrInternal)
	}

	responses := make([]responsedto.DebtResponseDto, 0, len(result.Data))
	for _, debt := range result.Data {
		responses = append(responses, toDebtResponse(debt))
	}

	// Cursor is nil on the last page; Encode is nil-safe (no panic).
	nextId, nextTime := result.Cursor.Encode()

	return &responsedto.DebtListResponseDto{
		AfterId:         nextId,
		AfterTime:       nextTime,
		HasNext:         result.HasNext,
		TransactionList: responses,
	}, nil
}

// GetDebtCustomer implements [domain.DebtUseCase].
func (d *debtUsecase) GetDebtCustomer(ctx context.Context, request *requestdto.GetDebtRequest) (*responsedto.DebtResponseDto, error) {
	id, err := uuid.Parse(request.DebtId)
	if err != nil {
		d.log.Error("failed to parse debt id", zap.Error(err))
		return nil, domain.InvalidID("invalid debt id format")
	}

	debt, err := d.debtRepo.GetDebtByID(ctx, id)
	if err != nil {
		d.log.Error("failed to get debt", zap.Error(err))
		if errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get debt: %w", domain.ErrInternal)
	}
	if debt == nil {
		d.log.Error("debt not found", zap.String("debt_id", request.DebtId))
		return nil, domain.NotFound("debt not found")
	}

	response := toDebtResponse(debt)
	return &response, nil
}

// PayDebtCash implements [domain.DebtUseCase].
// Records a cash payment the customer makes at the register toward an
// existing debt. The cashier (Flutter app) enters how much cash was received
// right now (request.NominalBayar) — it does not have to cover the full
// remaining balance. A nominal greater than what is still owed is rejected
// so the cashier can correct the amount before it's saved.
func (d *debtUsecase) PayDebtCash(ctx context.Context, request *requestdto.DebtPayment) (*responsedto.DebtPaymentResponse, error) {
	debtID, err := uuid.Parse(request.DebtID)
	if err != nil {
		d.log.Error("failed to parse debt id", zap.Error(err))
		return nil, domain.InvalidID("invalid debt id format")
	}
	userID, err := uuid.Parse(request.UserID)
	if err != nil {
		d.log.Error("failed to parse user id", zap.Error(err))
		return nil, domain.InvalidID("invalid user id format")
	}
	if request.NominalBayar <= 0 {
		return nil, fmt.Errorf("nominal_bayar must be greater than 0")
	}

	payment := &domain.DebtPayments{UserID: userID, NominalBayar: request.NominalBayar}
	result, err := d.debtRepo.PayDebt(ctx, debtID, payment)
	if err != nil {
		d.log.Error("failed to pay debt", zap.Error(err))
		if errors.Is(err, domain.ErrInternal) {
			return nil, fmt.Errorf("failed to record debt payment: %w", domain.ErrInternal)
		}
		// Business errors (not found, already paid off, overpayment) pass
		// through unwrapped so the cashier sees why it was rejected.
		return nil, err
	}

	return &responsedto.DebtPaymentResponse{
		DebtId:                result.Debt.ID.String(),
		CustomerName:          result.Debt.Customer.Name,
		NominalBayar:          request.NominalBayar,
		PreviousRemainingDebt: result.PreviousRemainingDebt,
		RemainingDebt:         result.Debt.RemainingDebt,
		TotalDebt:             result.Debt.TotalDebt,
		Status:                result.Debt.Status.String(),
		PaidAt:                result.PaidAt.Format(time.RFC3339),
	}, nil
}

// PrintReportDebtCustomer implements [domain.DebtUseCase].
// Builds a PDF debt report for a customer (summary + payment history)
// then returns the file URL the client can download.
func (d *debtUsecase) PrintReportDebtCustomer(ctx context.Context, request *requestdto.PrintDebtReport) (*responsedto.PrintDebtCustomerResponse, error) {
	if request.DebtId == "" {
		return nil, fmt.Errorf("debt_id is required")
	}
	id, err := uuid.Parse(request.DebtId)
	if err != nil {
		d.log.Error("failed to parse debt id", zap.Error(err))
		return nil, domain.InvalidID("invalid debt id format")
	}

	debt, err := d.debtRepo.GetDebtByID(ctx, id)
	if err != nil {
		d.log.Error("failed to get debt", zap.Error(err))
		if errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get debt: %w", domain.ErrInternal)
	}
	if debt == nil {
		d.log.Error("debt not found", zap.String("debt_id", request.DebtId))
		return nil, domain.NotFound("debt not found")
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
		return nil, fmt.Errorf("failed to generate report pdf: %w", domain.ErrInternal)
	}

	return &responsedto.PrintDebtCustomerResponse{
		CustomerName: debt.Customer.Name,
		DebtId:       debt.ID.String(),
		UrlPdf:       urlPdf,
	}, nil
}
