package usecase

import (
	"context"
	"shop_project_be/internal/domain"
	"shop_project_be/internal/dto/request"

	"go.uber.org/zap"
)

type transactionUsecase struct {
	trxRepo domain.TransactionRepository
	log     *zap.Logger
}

func NewTransactionUsecase(trxRepo domain.TransactionRepository, productRepo domain.ProductRepository, log *zap.Logger) domain.TransactionUsecase {
	return &transactionUsecase{
		trxRepo: trxRepo,
		log:     log,
	}
}

// AddTransaction implements [domain.TransactionUsecase].
func (t *transactionUsecase) AddTransaction(ctx context.Context, dto *request.AddTransactionRequest) error {
	panic("unimplemented")
}

// DeleteTransaction implements [domain.TransactionUsecase].
func (t *transactionUsecase) DeleteTransaction(ctx context.Context, dto *request.DeleteTransactionRequest) error {
	panic("unimplemented")
}

// GetAllTransaction implements [domain.TransactionUsecase].
func (t *transactionUsecase) GetAllTransaction(ctx context.Context, dto *request.FilterTransactionRequest) ([]domain.Transactions, error) {
	panic("unimplemented")
}

// GetTransaction implements [domain.TransactionUsecase].
func (t *transactionUsecase) GetTransaction(ctx context.Context, dto *request.GetTransactionRequest) (*domain.Transactions, error) {
	panic("unimplemented")
}

// PrintReportMonth implements [domain.TransactionUsecase].
func (t *transactionUsecase) PrintReportMonth(ctx context.Context, dto *request.PrintReportMonthRequest) (*domain.Transactions, error) {
	panic("unimplemented")
}

// PrintReportTransaction implements [domain.TransactionUsecase].
func (t *transactionUsecase) PrintReportTransaction(ctx context.Context, dto *request.PrintReportTransactionRequest) (*domain.Transactions, error) {
	panic("unimplemented")
}
