package domain

import (
	"context"
	"shop_project_be/internal/dto/request"
	"time"
)

type Debts struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CustomerID  uint      `gorm:"not null" json:"customer_id"`
	TotalHutang float64   `gorm:"type:decimal(15,2);not null" json:"total_hutang"`
	SisaHutang  float64   `gorm:"type:decimal(15,2);not null" json:"sisa_hutang"`
	Status      string    `gorm:"type:enum('belum_lunas','lunas');default:'belum_lunas'" json:"status"`
	JatuhTempo  time.Time `gorm:"column:jatuh_tempo" json:"jatuh_tempo"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	Customer     Customers      `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	Transactions []Transactions `gorm:"foreignKey:DebtID" json:"transactions,omitempty"`
	DebtPayments []DebtPayments `gorm:"foreignKey:DebtID" json:"payments"`
}

type DebtPayments struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	DebtID       uint      `gorm:"not null;index" json:"debt_id"`
	UserID       uint      `gorm:"not null" json:"user_id"`
	NominalBayar float64   `gorm:"type:decimal(15,2);not null" json:"nominal_bayar"`
	TanggalBayar time.Time `gorm:"autoCreateTime" json:"tanggal_bayar"`

	User *Users `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type DebtRepository interface {
	AddDebt(ctx *context.Context, debt *Debts) error
	DeleteDebt(ctx *context.Context, id uint) error
	GetAllDebt(ctx *context.Context) (*[]Debts, error)
}

type DebtUseCase interface {
	AddingDebtCustomer(ctx *context.Context, request *request.AddDebtRequest) error
	DeleteDebt(ctx *context.Context, request *request.DeleteDebtRequest) error
	GetAllDebtCustomerList(ctx *context.Context, request *request.GetAllDebtRequest) error
}
