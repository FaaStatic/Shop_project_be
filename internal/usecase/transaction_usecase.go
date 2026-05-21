package usecase

import (
	"context"
	"fmt"
	"shop_project_be/internal/constant/enum"
	"shop_project_be/internal/domain"
	requestdto "shop_project_be/internal/dto/request_dto"
	responsedto "shop_project_be/internal/dto/response_dto"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type transactionUsecase struct {
	trxRepo      domain.TransactionRepository
	productRepo  domain.ProductRepository
	userRepo     domain.UserRepository
	customerRepo domain.CustomerRepository
	debtRepo     domain.DebtRepository
	log          *zap.Logger
}

func NewTransactionUsecase(trxRepo domain.TransactionRepository, productRepo domain.ProductRepository, userRepo domain.UserRepository, customerRepo domain.CustomerRepository, debtRepo domain.DebtRepository, log *zap.Logger) domain.TransactionUsecase {
	return &transactionUsecase{
		trxRepo:      trxRepo,
		productRepo:  productRepo,
		userRepo:     userRepo,
		customerRepo: customerRepo,
		debtRepo:     debtRepo,
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
	panic("unimplemented")
}

// GetAllTransaction implements [domain.TransactionUsecase].
func (t *transactionUsecase) GetAllTransaction(ctx context.Context, dto *requestdto.FilterTransactionRequest) (*[]responsedto.TransactionResponse, error) {
	panic("unimplemented")
}

// GetTransaction implements [domain.TransactionUsecase].
func (t *transactionUsecase) GetTransaction(ctx context.Context, dto *requestdto.GetTransactionRequest) (*responsedto.TransactionResponse, error) {
	panic("unimplemented")
}

// PrintReportMonth implements [domain.TransactionUsecase].
func (t *transactionUsecase) PrintReportMonth(ctx context.Context, dto *requestdto.PrintReportMonthRequest) (*responsedto.PrintReportMonthTransactionResponse, error) {
	panic("unimplemented")
}

// PrintReportTransaction implements [domain.TransactionUsecase].
func (t *transactionUsecase) PrintReportTransaction(ctx context.Context, dto *requestdto.PrintReportTransactionRequest) (*responsedto.PrintReportTransactionResponse, error) {
	panic("unimplemented")
}
