package usecase

import (
	"context"
	"shop_project_be/internal/domain"
	requestdto "shop_project_be/internal/dto/request_dto"
	responsedto "shop_project_be/internal/dto/response_dto"

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
func (d *debtUsecase) AddingDebtCustomer(ctx context.Context, request *requestdto.AddDebtRequest) error {
	panic("unimplemented")
}

// DeleteDebtCustomer implements [domain.DebtUseCase].
func (d *debtUsecase) DeleteDebtCustomer(ctx context.Context, request *requestdto.DeleteDebtRequest) error {
	panic("unimplemented")
}

// GetAllDebtCustomerList implements [domain.DebtUseCase].
func (d *debtUsecase) GetAllDebtCustomerList(ctx context.Context, request *requestdto.FilterDebtRequest) (*[]responsedto.DebtResponseDto, error) {
	panic("unimplemented")
}

// GetDebtCustomer implements [domain.DebtUseCase].
func (d *debtUsecase) GetDebtCustomer(ctx context.Context, request *requestdto.GetDebtRequest) (*responsedto.DebtResponseDto, error) {
	panic("unimplemented")
}

// PrintReportDebtCustomer implements [domain.DebtUseCase].
func (d *debtUsecase) PrintReportDebtCustomer(ctx context.Context, request *requestdto.PrintDebtReport) (*responsedto.PrintDebtCustomerResponse, error) {
	panic("unimplemented")
}
