package domain

import (
	"context"
	"shop_project_be/internal/dto/request"
	"shop_project_be/internal/dto/response"
	"time"

	"github.com/google/uuid"
)

type Debts struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CustomerID  uuid.UUID `gorm:"type:uuid;not null" json:"customer_id"`
	TotalHutang float64   `gorm:"type:decimal(15,2);not null" json:"total_hutang"`
	SisaHutang  float64   `gorm:"type:decimal(15,2);not null" json:"sisa_hutang"`
	Status      string    `gorm:"type:enum('belum_lunas','lunas');default:'belum_lunas'" json:"status"`
	JatuhTempo  time.Time `gorm:"type:date" json:"jatuh_tempo"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	Customer     Customers      `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	Transactions []Transactions `gorm:"foreignKey:DebtID" json:"transactions,omitempty"`
	DebtPayments []DebtPayments `gorm:"foreignKey:DebtID" json:"payments"`
}

type DebtPayments struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	DebtID       uuid.UUID `gorm:"type:uuid;not null" json:"debt_id"`
	UserID       uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	NominalBayar float64   `gorm:"type:decimal(15,2);not null" json:"nominal_bayar"`
	TanggalBayar time.Time `gorm:"autoCreateTime" json:"tanggal_bayar"`

	User *Users `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type DebtRepository interface {
	AddDebt(ctx *context.Context, debt *Debts) error
	DeleteDebt(ctx *context.Context, id uuid.UUID) error
	GetAllDebt(ctx *context.Context) (*[]Debts, error)
}

type DebtUseCase interface {
	AddingDebtCustomer(ctx *context.Context, request *request.AddDebtRequest) error
	DeleteDebtCustomer(ctx *context.Context, request *request.DeleteDebtRequest) error
	GetAllDebtCustomerList(ctx *context.Context, request *request.FilterDebtRequest) (*[]response.DebtResponseDto, error)
	GetDebtCustomer(ctx *context.Context, request *request.GetDebtRequest) (*response.DebtResponseDto, error)
	PrintReportDebtCustomer(ctx *context.Context, request *request.PrintDebtReport) (*response.PrintDebtCustomerResponse, error)
}
