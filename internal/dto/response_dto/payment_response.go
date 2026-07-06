package responsedto

// ChargePaymentResponse is the response after a charge is successfully created. For
// QRIS, Flutter shows QrUrl/QrString as a QR code. For cards, if
// 3DS is needed, Flutter opens RedirectUrl in a webview. The final status must NOT
// rely on this response alone — wait for the webhook/check status (see notes).
type ChargePaymentResponse struct {
	OrderID     string `json:"order_id"`     // = no_invoice
	Method      string `json:"method"`       // "qris" | "card"
	Status      string `json:"status"`       // status internal: pending/success/failed/expired
	GrossAmount int64  `json:"gross_amount"` // in rupiah (no decimals)

	MidtransStatus string `json:"midtrans_status"`        // settlement/pending/capture/...
	QrString       string `json:"qr_string,omitempty"`    // payload QR (alternatif render manual)
	QrUrl          string `json:"qr_url,omitempty"`       // URL gambar QR siap tampil (QRIS)
	RedirectUrl    string `json:"redirect_url,omitempty"` // URL 3DS (kartu)
	ExpiryTime     string `json:"expiry_time,omitempty"`  // batas waktu bayar
}

// PaymentStatusResponse is used by Flutter to poll the payment status.
// no_invoice = order_id; when Status is "success", transaction_id is populated.
type PaymentStatusResponse struct {
	OrderID        string `json:"order_id"`
	Method         string `json:"method"`
	Status         string `json:"status"`          // pending | success | failed | expired
	MidtransStatus string `json:"midtrans_status"` // raw Midtrans status
	GrossAmount    int64  `json:"gross_amount"`
	TransactionID  string `json:"transaction_id,omitempty"`
	PaidAt         string `json:"paid_at,omitempty"`
}
