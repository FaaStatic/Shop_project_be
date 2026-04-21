package usecase

import (
	"context"
	"shop_project_be/internal/domain"
	"shop_project_be/internal/dto/request"
	"shop_project_be/internal/dto/response"

	"go.uber.org/zap"
)

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

// AddingDebtCustomer implements [domain.DebtUseCase].
func (d *debtUsecase) AddingDebtCustomer(ctx context.Context, request *request.AddDebtRequest) error {
	panic("unimplemented")
}

// DeleteDebtCustomer implements [domain.DebtUseCase].
func (d *debtUsecase) DeleteDebtCustomer(ctx context.Context, request *request.DeleteDebtRequest) error {
	panic("unimplemented")
}

// GetAllDebtCustomerList implements [domain.DebtUseCase].
func (d *debtUsecase) GetAllDebtCustomerList(ctx context.Context, request *request.FilterDebtRequest) (*[]response.DebtResponseDto, error) {
	panic("unimplemented")
}

// GetDebtCustomer implements [domain.DebtUseCase].
func (d *debtUsecase) GetDebtCustomer(ctx context.Context, request *request.GetDebtRequest) (*response.DebtResponseDto, error) {
	panic("unimplemented")
}

// PrintReportDebtCustomer implements [domain.DebtUseCase].
func (d *debtUsecase) PrintReportDebtCustomer(ctx context.Context, request *request.PrintDebtReport) (*response.PrintDebtCustomerResponse, error) {
	panic("unimplemented")
}
