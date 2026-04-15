package domain

import (
	"context"
	"shop_project_be/internal/dto/request"
	"time"

	"github.com/google/uuid"
)

type Transactions struct {
	ID         uint   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	NoInvoice  string `gorm:"type:varchar(50);uniqueIndex;not null" json:"no_invoice"`
	UserID     uint   `gorm:"type:uuid;not null" json:"user_id"`
	CustomerID *uint  `gorm:"column:customer_id" json:"customer_id"`
	DebtID     *uint  `gorm:"column:debt_id;index" json:"debt_id"`

	TipePembayaran string    `gorm:"type:enum('tunai','hutang','transfer','qris');not null" json:"tipe_pembayaran"`
	TotalTransaksi float64   `gorm:"type:decimal(15,2);not null" json:"total_transaksi"`
	TotalLaba      float64   `gorm:"type:decimal(15,2);not null" json:"total_laba"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`

	User              Users                `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Customer          Customers            `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	TransactionDetail []TransactionsDetail `gorm:"foreignKey:TransactionID" json:"details"`
}

type TransactionsDetail struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TransactionID uuid.UUID `gorm:"type:uuid;not null" json:"transaction_id"`
	ProductID     uuid.UUID `gorm:"type:uuid;not null" json:"product_id"`
	Qty           float64   `gorm:"type:decimal(8,2);not null" json:"qty"`
	Subtotal      float64   `gorm:"type:decimal(15,2);not null" json:"subtotal"`

	Product Products `gorm:"foreignKey:ProductID" json:"product,omitempty"`
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
