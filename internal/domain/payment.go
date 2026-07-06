package domain

import (
	"context"
	"errors"
	requestdto "shop_project_be/internal/dto/request_dto"
	responsedto "shop_project_be/internal/dto/response_dto"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PaymentStatus is the internal payment status (not the raw Midtrans status).
type PaymentStatus string

const (
	PaymentPending PaymentStatus = "pending"
	PaymentSuccess PaymentStatus = "success"
	PaymentFailed  PaymentStatus = "failed"
	PaymentExpired PaymentStatus = "expired"
)

// ErrPaymentAccessDenied is returned when a user tries to access the payment
// status of another user without the admin/superadmin role.
var ErrPaymentAccessDenied = errors.New("payment access denied")

// PaymentItem is a cart line stored together with the payment. When the
// payment succeeds, this item is used to create the transaction (deduct stock +
// compute price on the server). Intentionally only product_id + qty; the price
// is recomputed server-side so it cannot be manipulated by the client.
type PaymentItem struct {
	ProductID uuid.UUID `json:"product_id"`
	Qty       float64   `json:"qty"`
}

// Payment is the lifecycle record of a payment via Midtrans. OrderID is used
// as the order_id in Midtrans and also as the no_invoice in the transactions table.
type Payment struct {
	ID          uuid.UUID     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrderID     string        `gorm:"type:varchar(50);uniqueIndex;not null" json:"order_id"`
	UserID      uuid.UUID     `gorm:"type:uuid;not null" json:"user_id"`
	CustomerID  *uuid.UUID    `gorm:"type:uuid" json:"customer_id,omitempty"`
	Method      string        `gorm:"type:varchar(20);not null" json:"method"` // "qris" | "va"
	GrossAmount float64       `gorm:"type:decimal(15,2);not null" json:"gross_amount"`
	Status      PaymentStatus `gorm:"type:varchar(20);not null;default:'pending'" json:"status"`

	// Raw data from Midtrans for audit/reconciliation.
	MidtransTrxID  string `gorm:"column:midtrans_trx_id;type:varchar(100)" json:"midtrans_transaction_id"`
	MidtransStatus string `gorm:"column:midtrans_status;type:varchar(30)" json:"midtrans_status"`
	FraudStatus    string `gorm:"column:fraud_status;type:varchar(30)" json:"fraud_status"`
	QRString       string `gorm:"column:qr_string;type:text" json:"qr_string"`
	QRURL          string `gorm:"column:qr_url;type:text" json:"qr_url"`
	RedirectURL    string `gorm:"column:redirect_url;type:text" json:"redirect_url"`

	// VA (Virtual Account) details, filled for method == "va".
	VABank     string `gorm:"column:va_bank;type:varchar(20)" json:"va_bank,omitempty"`     // "bca"|"mandiri"
	VANumber   string `gorm:"column:va_number;type:varchar(50)" json:"va_number,omitempty"` // BCA bank_transfer VA
	BillKey    string `gorm:"column:bill_key;type:varchar(50)" json:"bill_key,omitempty"`   // Mandiri echannel
	BillerCode string `gorm:"column:biller_code;type:varchar(20)" json:"biller_code,omitempty"`

	Items         []PaymentItem `gorm:"serializer:json;type:jsonb" json:"items"`
	ExpiryTime    *time.Time    `json:"expiry_time,omitempty"`
	PaidAt        *time.Time    `json:"paid_at,omitempty"`
	TransactionID *uuid.UUID    `gorm:"type:uuid" json:"transaction_id,omitempty"` // filled when the transaction is created

	// StockReserved is true while the item stock is still reserved for this
	// payment: deducted when the charge is created, returned when it lapses, and
	// consumed (flag cleared without restore) when the payment succeeds.
	StockReserved bool `gorm:"column:stock_reserved;not null;default:false" json:"stock_reserved"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (p *Payment) TableName() string {
	return "payments"
}

// --- Port to the payment gateway (Midtrans). Defined in the domain so the
// usecase does not depend directly on the SDK; implemented in infrastructure. ---

// GatewayItem, GatewayCustomer, GatewayChargeInput are gateway-neutral types
// (containing no type from any SDK).
type GatewayItem struct {
	ID    string
	Name  string
	Price int64
	Qty   int32
}

type GatewayCustomer struct {
	FirstName string
	Email     string
	Phone     string
}

type GatewayChargeInput struct {
	OrderID     string
	GrossAmount int64
	Items       []GatewayItem
	Customer    GatewayCustomer

	// VA-only: "bca" (bank_transfer) or "mandiri" (echannel).
	Bank string
}

// GatewayChargeResult is the normalized charge/status result.
type GatewayChargeResult struct {
	TransactionID     string
	OrderID           string
	PaymentType       string
	TransactionStatus string // settlement | capture | pending | deny | expire | cancel
	FraudStatus       string // accept | challenge | deny
	StatusCode        string
	QRString          string
	QRURL             string
	RedirectURL       string
	ExpiryTime        string

	VANumber   string
	Bank       string
	BillKey    string
	BillerCode string
}

type PaymentGateway interface {
	ChargeQris(ctx context.Context, in GatewayChargeInput) (*GatewayChargeResult, error)
	ChargeVA(ctx context.Context, in GatewayChargeInput) (*GatewayChargeResult, error)
	CheckStatus(ctx context.Context, orderID string) (*GatewayChargeResult, error)
	// VerifySignature validates the Midtrans notification signature_key:
	// SHA512(order_id + status_code + gross_amount + ServerKey).
	VerifySignature(orderID, statusCode, grossAmount, signatureKey string) bool
}

type PaymentRepository interface {
	Create(ctx context.Context, payment *Payment) error
	GetByOrderID(ctx context.Context, orderID string) (*Payment, error)
	Update(ctx context.Context, payment *Payment) error
	// UpdateWithLock reads the payment locked FOR UPDATE, calls fn, then saves
	// the changes if fn returns save=true — all within a single DB transaction.
	// Used by the webhook flow so concurrent notifications for the same order are
	// serialized (no double-finalize / double push).
	UpdateWithLock(ctx context.Context, orderID string, fn func(payment *Payment) (save bool, err error)) error
	// ListStalePending returns pending payments past expiry_time (or older than
	// 24h if expiry is not recorded) — reconciliation candidates because their
	// webhook never arrived or kept failing.
	ListStalePending(ctx context.Context, limit int) ([]*Payment, error)
}

type PaymentUsecase interface {
	ChargeQris(ctx context.Context, request *requestdto.ChargeQrisRequest) (*responsedto.ChargePaymentResponse, error)
	ChargeCard(ctx context.Context, request *requestdto.ChargeCardRequest) (*responsedto.ChargePaymentResponse, error)
	HandleNotification(ctx context.Context, notif *requestdto.MidtransNotificationRequest) error
	// GetStatus returns the payment status. requesterID/requesterRole are used
	// for the ownership check: besides admin/superadmin, only the order creator
	// may view its status (ErrPaymentAccessDenied otherwise).
	GetStatus(ctx context.Context, orderID, requesterID, requesterRole string) (*responsedto.PaymentStatusResponse, error)
	// ReconcileStalePayments sweeps expired pending payments: ask Midtrans for the
	// authoritative status then apply it (release stock reservation / finalize).
	ReconcileStalePayments(ctx context.Context) error
}
