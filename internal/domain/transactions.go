package domain

import (
	"context"
	"shop_project_be/internal/constant/enum"
	"shop_project_be/internal/dto/request"
	"time"

	"github.com/google/uuid"
)

type Transactions struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	NoInvoice  string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"no_invoice"`
	UserID     uuid.UUID  `gorm:"type:uuid;not null" json:"user_id"`
	CustomerID *uuid.UUID `gorm:"column:customer_id" json:"customer_id"`
	DebtID     *uuid.UUID `gorm:"column:debt_id;index" json:"debt_id"`

	PaymentType      enum.MoneyPayment `gorm:"type:smallint;check:payment_type IN (0,1,2,3);not null" json:"payment_type"`
	TotalTransaction float64           `gorm:"type:decimal(15,2);not null" json:"total_transaction"`
	TotalProfit      float64           `gorm:"type:decimal(15,2);not null" json:"total_profit"`
	CreatedAt        time.Time         `gorm:"autoCreateTime" json:"created_at"`

	User              Users                `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Customer          Customers            `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	TransactionDetail []TransactionsDetail `gorm:"foreignKey:TransactionID" json:"details"`
}

type TransactionsDetail struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TransactionID uuid.UUID `gorm:"type:uuid;not null" json:"transaction_id"`
	ProductID     uuid.UUID `gorm:"type:uuid;not null" json:"product_id"`
	Price         float64   `gorm:"type:decimal(15,2);not null" json:"price"`
	PriceDebt     float64   `gorm:"type:decimal(15,2);not null" json:"price_debt"`
	Qty           float64   `gorm:"type:decimal(8,2);not null" json:"qty"`
	Subtotal      float64   `gorm:"type:decimal(15,2);not null" json:"subtotal"`

	Product Products `gorm:"foreignKey:ProductID" json:"product,omitempty"`
}

func (t *Transactions) TableName() string {
	return "transactions"
}

func (td *TransactionsDetail) TableName() string {
	return "transactions_detail"
}

type TransactionRepository interface {
	CreateTransaction(ctx *context.Context, transaction *Transactions) error
	GetTransactionByID(ctx *context.Context, id uuid.UUID) (*Transactions, error)
	GetAllTransaction(ctx *context.Context) (*[]Transactions, error)
	DeleteTransaction(ctx *context.Context, id uuid.UUID) error
	UpdateTransaction(ctx *context.Context, id uuid.UUID, trx *Transactions) error
}

type TransactionUsecase interface {
	AddTransaction(ctx *context.Context, transactionDto *request.AddTransactionDetailRequest) error
	GetTransaction(ctx *context.Context, transactionDto *request.GetTransactionRequest) (*Transactions, error)
	GetAllTransaction(ctx *context.Context, transactionDto *request.FilterTransactionRequest) (*[]Transactions, error)
	DeleteTransaction(ctx *context.Context, transactionDto *request.DeleteTransactionRequest) error
	PrintReportTransaction(ctx *context.Context, transactionDto *request.PrintReportTransactionRequest) (*Transactions, error)
	PrintReportMonth(ctx *context.Context, transactionDto *request.PrintReportMonthRequest) (*Transactions, error)
}
