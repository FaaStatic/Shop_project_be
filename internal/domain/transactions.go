package domain

import (
	"context"
	"shop_project_be/internal/constant/enum"
	"shop_project_be/internal/constant/paginated"
	requestdto "shop_project_be/internal/dto/request_dto"
	responsedto "shop_project_be/internal/dto/response_dto"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Transactions struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	NoInvoice  string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"no_invoice"`
	UserID     uuid.UUID  `gorm:"type:uuid;not null" json:"user_id"`
	CustomerID *uuid.UUID `gorm:"column:customer_id" json:"customer_id"`
	DebtID     *uuid.UUID `gorm:"column:debt_id;index" json:"debt_id"`

	PaymentType      enum.MoneyPayment `gorm:"type:smallint;check:payment_type IN (0,1,2,3);not null" json:"payment_type"`
	Bank             *string           `gorm:"type:varchar(20)" json:"bank,omitempty"` // "bca"|"mandiri", set only when PaymentType == transfer
	TotalTransaction float64           `gorm:"type:decimal(15,2);not null" json:"total_transaction"`
	CreatedAt        time.Time         `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time         `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt        gorm.DeletedAt    `gorm:"index" json:"-"`

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
	Destination   *string   `gorm:"type:varchar(50)" json:"destination,omitempty"` // phone/e-wallet account for digital products

	Product Products `gorm:"foreignKey:ProductID" json:"product,omitempty"`
}

func (t *Transactions) TableName() string {
	return "transactions"
}

func (td *TransactionsDetail) TableName() string {
	return "transactions_detail"
}

type FilterTransaction struct {
	NoInvoices string
	Cursor     *paginated.CursorMeta
	DateStart  *string
	DateEnd    *string
	Limit      int
	TypeTrx    *int
	Order      string
}

type ResultTransaction struct {
	DataItem []*Transactions
	HasNext  bool
	Cursor   *paginated.CursorMeta
}

// MonthlyReport is the aggregation of transactions over one month.
type MonthlyReport struct {
	TotalTransaction int64   `gorm:"column:total_transaction"` // number of transactions
	TotalRevenue     float64 `gorm:"column:total_revenue"`     // incoming revenue (excluding debt)
	TotalDebt        float64 `gorm:"column:total_debt"`        // value of debt transactions
	GrandTotal       float64 `gorm:"column:grand_total"`       // total of all transaction values
}

// DailyReport is the aggregation of transactions on a single day of that month.
type DailyReport struct {
	Date             time.Time `gorm:"column:date"`
	TotalTransaction int64     `gorm:"column:total_transaction"`
	TotalRevenue     float64   `gorm:"column:total_revenue"`
	TotalDebt        float64   `gorm:"column:total_debt"`
	GrandTotal       float64   `gorm:"column:grand_total"`
}

// ProductSoldReport is the recap of a single product sold during a month.
type ProductSoldReport struct {
	ProductName string  `gorm:"column:product_name"`
	Qty         float64 `gorm:"column:qty"`
	Total       float64 `gorm:"column:total"`
}

// DailyProductSoldReport is the recap of a single product sold on a single day.
type DailyProductSoldReport struct {
	Date        time.Time `gorm:"column:date"`
	ProductName string    `gorm:"column:product_name"`
	Qty         float64   `gorm:"column:qty"`
	Total       float64   `gorm:"column:total"`
}

// TransactionDebtSnapshot captures how a hutang (debt) sale affected the
// customer's debt balance, everything a receipt needs to show "sisa hutang
// sebelumnya" vs "sisa hutang sekarang". CreateTransaction only returns this
// when the sale is a debt sale linked to a customer; a cash/non-debt sale
// returns nil since it never touches the debts table.
type TransactionDebtSnapshot struct {
	DebtID                uuid.UUID
	PreviousRemainingDebt float64         // remaining debt before this transaction
	AmountAdded           float64         // this transaction's total, added to the debt
	TotalDebt             float64         // cumulative total ever owed, after this transaction
	RemainingDebt         float64         // remaining owed, after this transaction
	Status                enum.DebtStatus // BELUM_LUNAS/LUNAS after this transaction
}

type TransactionRepository interface {
	// CreateTransaction saves the transaction + details atomically. deductStock
	// is false for transactions from online payments whose stock was already
	// reserved at charge time (must not be deducted twice). Returns a
	// TransactionDebtSnapshot only when isHutang is true and the transaction is
	// linked to a customer (nil for cash/non-debt sales).
	CreateTransaction(ctx context.Context, transaction *Transactions, isHutang bool, deductStock bool) (*TransactionDebtSnapshot, error)
	GetTransactionByID(ctx context.Context, id uuid.UUID) (*Transactions, error)
	GetAllTransaction(ctx context.Context, filter FilterTransaction) (*ResultTransaction, error)
	DeleteTransaction(ctx context.Context, id uuid.UUID) error
	UpdateTransaction(ctx context.Context, id uuid.UUID, trx *Transactions) error
	CheckTransactionByNoInvoice(ctx context.Context, noInvoice string) (*Transactions, error)
	GetMonthlyReport(ctx context.Context, month int, year int) (*MonthlyReport, error)
	GetDailyReport(ctx context.Context, month int, year int) ([]DailyReport, error)
	GetMonthlyProductSold(ctx context.Context, month int, year int) ([]ProductSoldReport, error)
	GetDailyProductSold(ctx context.Context, month int, year int) ([]DailyProductSoldReport, error)
}

type TransactionUsecase interface {
	// AddTransaction returns an AddTransactionResponse whose DebtInfo is set
	// only when the sale is hutang (debt); a cash/transfer/qris sale leaves
	// DebtInfo nil.
	AddTransaction(ctx context.Context, dto *requestdto.AddTransactionRequest) (*responsedto.AddTransactionResponse, error)
	// AddPrepaidTransaction is like AddTransaction but does NOT deduct stock —
	// only for transactions from online payments whose stock was already
	// reserved. Do not expose it to the HTTP handler.
	AddPrepaidTransaction(ctx context.Context, dto *requestdto.AddTransactionRequest) (*responsedto.AddTransactionResponse, error)
	GetTransaction(ctx context.Context, dto *requestdto.GetTransactionRequest) (*responsedto.TransactionResponse, error)
	GetAllTransaction(ctx context.Context, dto *requestdto.FilterTransactionRequest) (*responsedto.GetAllTransactionResponse, error)
	DeleteTransaction(ctx context.Context, dto *requestdto.DeleteTransactionRequest) error
	PrintReportTransaction(ctx context.Context, dto *requestdto.PrintReportTransactionRequest) (*responsedto.PrintReportTransactionResponse, error)
	PrintReportMonth(ctx context.Context, dto *requestdto.PrintReportMonthRequest) (*responsedto.PrintReportMonthTransactionResponse, error)
}
