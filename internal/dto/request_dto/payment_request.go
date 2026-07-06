package requestdto

// PaymentItemRequest is a single cart line. The price is NOT sent by the client;
// it is computed server-side from the product price so it cannot be manipulated.
type PaymentItemRequest struct {
	ProductId string  `json:"product_id" validate:"required,uuid"`
	Qty       float64 `json:"qty" validate:"required,gt=0"`
}

// ChargeQrisRequest is used by Flutter to request a QRIS payment. no_invoice
// is optional; if empty, the server generates a unique invoice number.
type ChargeQrisRequest struct {
	UserId     string               `json:"user_id" validate:"required,uuid"`
	CustomerId *string              `json:"customer_id,omitempty" validate:"omitempty,uuid"`
	NoInvoice  string               `json:"no_invoice,omitempty"`
	Items      []PaymentItemRequest `json:"items" validate:"required,min=1,dive"`
}

// ChargeVARequest is used by Flutter for Virtual Account payments. Bank selects
// the VA channel: "bca" (bank_transfer) or "mandiri" (echannel).
type ChargeVARequest struct {
	UserId     string               `json:"user_id" validate:"required,uuid"`
	CustomerId *string              `json:"customer_id,omitempty" validate:"omitempty,uuid"`
	NoInvoice  string               `json:"no_invoice,omitempty"`
	Bank       string               `json:"bank" validate:"required,oneof=bca mandiri"`
	Items      []PaymentItemRequest `json:"items" validate:"required,min=1,dive"`
}

// MidtransNotificationRequest is the HTTP webhook payload from Midtrans. Only
// the fields needed for verification & status update are mapped.
type MidtransNotificationRequest struct {
	OrderID           string `json:"order_id"`
	StatusCode        string `json:"status_code"`
	GrossAmount       string `json:"gross_amount"`
	SignatureKey      string `json:"signature_key"`
	TransactionStatus string `json:"transaction_status"`
	FraudStatus       string `json:"fraud_status"`
	PaymentType       string `json:"payment_type"`
	TransactionID     string `json:"transaction_id"`
}
