package usecase

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"shop_project_be/internal/domain"
	requestdto "shop_project_be/internal/dto/request_dto"
	responsedto "shop_project_be/internal/dto/response_dto"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type paymentUsecase struct {
	paymentRepo domain.PaymentRepository
	gateway     domain.PaymentGateway
	productRepo domain.ProductRepository
	trxUsecase  domain.TransactionUsecase
	trxRepo     domain.TransactionRepository
	notifier    domain.DeviceTokenUsecase
	log         *zap.Logger
}

func NewPaymentUsecase(
	paymentRepo domain.PaymentRepository,
	gateway domain.PaymentGateway,
	productRepo domain.ProductRepository,
	trxUsecase domain.TransactionUsecase,
	trxRepo domain.TransactionRepository,
	notifier domain.DeviceTokenUsecase,
	log *zap.Logger,
) domain.PaymentUsecase {
	return &paymentUsecase{
		paymentRepo: paymentRepo,
		gateway:     gateway,
		productRepo: productRepo,
		trxUsecase:  trxUsecase,
		trxRepo:     trxRepo,
		notifier:    notifier,
		log:         log,
	}
}

// ChargeQris creates a QRIS payment then returns the QR to display in
// Flutter. The final status arrives later via webhook (see HandleNotification).
func (u *paymentUsecase) ChargeQris(ctx context.Context, request *requestdto.ChargeQrisRequest) (*responsedto.ChargePaymentResponse, error) {
	userID, err := uuid.Parse(request.UserId)
	if err != nil {
		u.log.Error("failed to parse user id", zap.Error(err))
		return nil, fmt.Errorf("invalid user id format")
	}
	customerID, err := parseOptionalUUID(request.CustomerId)
	if err != nil {
		u.log.Error("failed to parse customer id", zap.Error(err))
		return nil, fmt.Errorf("invalid customer id format")
	}

	gross, items, err := u.buildOrder(ctx, toItemPairs(request.Items))
	if err != nil {
		return nil, err
	}
	gross = applyFee("qris", gross)

	orderID, err := u.resolveOrderID(ctx, request.NoInvoice)
	if err != nil {
		return nil, err
	}

	// Reserve stock BEFORE charging: once the QR is issued the goods are already
	// allocated — two buyers cannot both pay for the last unit.
	if err := u.productRepo.ReserveStock(ctx, items); err != nil {
		u.log.Warn("failed to reserve stock", zap.Error(err), zap.String("order_id", orderID))
		return nil, fmt.Errorf("insufficient stock")
	}

	result, err := u.gateway.ChargeQris(ctx, domain.GatewayChargeInput{
		OrderID:     orderID,
		GrossAmount: gross,
	})
	if err != nil {
		u.releaseStock(ctx, items, orderID) // charge failed to create: return the reservation
		u.log.Error("failed to charge qris", zap.Error(err))
		return nil, fmt.Errorf("failed to create qris payment")
	}

	payment := u.newPayment(orderID, "qris", userID, customerID, gross, items, result)
	payment.StockReserved = true
	return u.persistChargeResult(ctx, payment, result)
}

// ChargeVA creates a Virtual Account payment (BCA bank_transfer or Mandiri
// echannel). The buyer pays the displayed VA number / bill; the final status
// arrives via webhook (see HandleNotification).
func (u *paymentUsecase) ChargeVA(ctx context.Context, request *requestdto.ChargeVARequest) (*responsedto.ChargePaymentResponse, error) {
	userID, err := uuid.Parse(request.UserId)
	if err != nil {
		u.log.Error("failed to parse user id", zap.Error(err))
		return nil, fmt.Errorf("invalid user id format")
	}
	customerID, err := parseOptionalUUID(request.CustomerId)
	if err != nil {
		u.log.Error("failed to parse customer id", zap.Error(err))
		return nil, fmt.Errorf("invalid customer id format")
	}

	gross, items, err := u.buildOrder(ctx, toItemPairs(request.Items))
	if err != nil {
		return nil, err
	}
	gross = applyFee("va", gross)

	orderID, err := u.resolveOrderID(ctx, request.NoInvoice)
	if err != nil {
		return nil, err
	}

	// Reserve stock BEFORE charging (see ChargeQris).
	if err := u.productRepo.ReserveStock(ctx, items); err != nil {
		u.log.Warn("failed to reserve stock", zap.Error(err), zap.String("order_id", orderID))
		return nil, fmt.Errorf("insufficient stock")
	}

	result, err := u.gateway.ChargeVA(ctx, domain.GatewayChargeInput{
		OrderID:     orderID,
		GrossAmount: gross,
		Bank:        request.Bank,
	})
	if err != nil {
		u.releaseStock(ctx, items, orderID)
		u.log.Error("failed to charge va", zap.Error(err))
		return nil, fmt.Errorf("failed to create va payment")
	}

	payment := u.newPayment(orderID, "va", userID, customerID, gross, items, result)
	payment.StockReserved = true
	return u.persistChargeResult(ctx, payment, result)
}

// HandleNotification processes the Midtrans webhook: verify signature, fetch
// the authoritative status, update the payment, and on success create the transaction
// (carrying no_invoice into the transactions table). Idempotent against repeated
// notifications sent by Midtrans.
func (u *paymentUsecase) HandleNotification(ctx context.Context, notif *requestdto.MidtransNotificationRequest) error {
	// Log EVERY incoming notification (proof Midtrans called) — before
	// verification, so even attempts with a wrong signature remain visible.
	u.log.Info("midtrans notification received",
		zap.String("order_id", notif.OrderID),
		zap.String("transaction_status", notif.TransactionStatus),
		zap.String("fraud_status", notif.FraudStatus),
		zap.String("payment_type", notif.PaymentType),
	)

	if !u.gateway.VerifySignature(notif.OrderID, notif.StatusCode, notif.GrossAmount, notif.SignatureKey) {
		u.log.Warn("invalid midtrans signature", zap.String("order_id", notif.OrderID))
		return domain.ErrInvalidSignature
	}

	payment, err := u.paymentRepo.GetByOrderID(ctx, notif.OrderID)
	if err != nil {
		u.log.Error("failed to get payment", zap.Error(err))
		return fmt.Errorf("failed to get payment")
	}
	if payment == nil {
		// Unknown order: still reply success (200) so Midtrans stops retrying.
		u.log.Warn("notification for unknown order", zap.String("order_id", notif.OrderID))
		return nil
	}
	if isFinalPaymentStatus(payment.Status) {
		// Already final (success/failed/expired): Midtrans replays are ignored so
		// data is not rewritten and the push notification is not sent twice.
		return nil
	}

	// Fetch the authoritative status; the notification payload can be replayed. Done
	// outside the lock — do not hold the row lock during an outgoing HTTP call.
	trxStatus, fraudStatus, midtransTrxID := notif.TransactionStatus, notif.FraudStatus, notif.TransactionID
	if authoritative, cerr := u.gateway.CheckStatus(ctx, notif.OrderID); cerr == nil {
		trxStatus, fraudStatus, midtransTrxID = authoritative.TransactionStatus, authoritative.FraudStatus, authoritative.TransactionID
	} else {
		u.log.Warn("failed to verify status to midtrans, fallback to payload", zap.Error(cerr))
	}

	return u.applyAuthoritativeStatus(ctx, notif.OrderID, trxStatus, fraudStatus, midtransTrxID)
}

// applyAuthoritativeStatus applies the Midtrans status to the payment under a row
// lock (two concurrent processors are serialized; the loser sees the final status
// and stops), releases the stock reservation if the payment lapses, then sends
// the push notification. Used by the webhook and the reconciliation job.
func (u *paymentUsecase) applyAuthoritativeStatus(ctx context.Context, orderID, trxStatus, fraudStatus, midtransTrxID string) error {
	var final *domain.Payment
	var releaseItems []domain.PaymentItem // reservation returned after commit
	err := u.paymentRepo.UpdateWithLock(ctx, orderID, func(p *domain.Payment) (bool, error) {
		if isFinalPaymentStatus(p.Status) {
			return false, nil // already finalized by another processor
		}
		p.MidtransStatus = trxStatus
		p.FraudStatus = fraudStatus
		if midtransTrxID != "" {
			p.MidtransTrxID = midtransTrxID
		}

		switch mapInternalStatus(trxStatus, fraudStatus) {
		case domain.PaymentSuccess:
			if err := u.finalizeSuccess(ctx, p); err != nil {
				// Money already settled but the transaction failed to create — needs
				// manual follow-up (refund via the Midtrans dashboard /
				// stock correction). This marker is what alerting watches.
				u.log.Error("PAYMENT_RECONCILIATION_REQUIRED: settled payment cannot be finalized",
					zap.Error(err), zap.String("order_id", p.OrderID))
				return false, fmt.Errorf("failed to finalize successful payment: %w", err)
			}
		case domain.PaymentFailed:
			p.Status = domain.PaymentFailed
		case domain.PaymentExpired:
			p.Status = domain.PaymentExpired
		default:
			p.Status = domain.PaymentPending
		}

		// Payment lapsed: the flag is cleared with the commit (so replays don't
		// double-restore); stock is returned after commit, outside the lock.
		if p.StockReserved && (p.Status == domain.PaymentFailed || p.Status == domain.PaymentExpired) {
			releaseItems = p.Items
			p.StockReserved = false
		}
		final = p
		return true, nil
	})
	if err != nil {
		u.log.Error("failed to process payment status", zap.Error(err), zap.String("order_id", orderID))
		return fmt.Errorf("failed to update payment")
	}
	if final == nil {
		return nil // handled by another flow; no one to notify
	}

	if len(releaseItems) > 0 {
		u.releaseStock(ctx, releaseItems, final.OrderID)
	}

	// Push notification only on transition to a final status; pending does not
	// need notifying. Sent AFTER commit so it does not precede the data.
	switch final.Status {
	case domain.PaymentSuccess:
		u.notifyPaymentResult(ctx, final, true)
	case domain.PaymentFailed, domain.PaymentExpired:
		u.notifyPaymentResult(ctx, final, false)
	}

	u.log.Info("payment status processed",
		zap.String("order_id", final.OrderID),
		zap.String("midtrans_status", final.MidtransStatus),
		zap.String("internal_status", string(final.Status)),
	)
	return nil
}

// ReconcileStalePayments sweeps pending payments past their validity period
// (webhook never arrived or kept failing): ask Midtrans for the authoritative
// status then apply it — expired ones get their stock reservation released, settled
// ones are finalized. Called periodically from the scheduler in cmd.
func (u *paymentUsecase) ReconcileStalePayments(ctx context.Context) error {
	stale, err := u.paymentRepo.ListStalePending(ctx, 50)
	if err != nil {
		return fmt.Errorf("failed to list stale payments: %w", err)
	}
	for _, p := range stale {
		authoritative, err := u.gateway.CheckStatus(ctx, p.OrderID)
		if err != nil {
			// Without an authoritative status don't guess — leave it for the next
			// sweep rather than wrongly releasing stock for a paid payment.
			u.log.Warn("reconcile: failed to check status", zap.Error(err), zap.String("order_id", p.OrderID))
			continue
		}
		if err := u.applyAuthoritativeStatus(ctx, p.OrderID, authoritative.TransactionStatus, authoritative.FraudStatus, authoritative.TransactionID); err != nil {
			u.log.Error("reconcile: failed to apply status", zap.Error(err), zap.String("order_id", p.OrderID))
		}
	}
	return nil
}

// isFinalPaymentStatus marks statuses that must not be reprocessed by
// notification replays — preventing double updates and duplicate push notifications.
func isFinalPaymentStatus(s domain.PaymentStatus) bool {
	switch s {
	case domain.PaymentSuccess, domain.PaymentFailed, domain.PaymentExpired:
		return true
	}
	return false
}

// GetStatus returns the current payment status from the DB (the source of truth
// already updated by the webhook). Used by Flutter for polling. Besides
// admin/superadmin, only the order creator may view its status; every
// access is recorded as an audit log (to detect snooping between staff).
func (u *paymentUsecase) GetStatus(ctx context.Context, orderID, requesterID, requesterRole string) (*responsedto.PaymentStatusResponse, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return nil, fmt.Errorf("order_id is required")
	}
	payment, err := u.paymentRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		u.log.Error("failed to get payment", zap.Error(err))
		return nil, fmt.Errorf("failed to get payment")
	}
	if payment == nil {
		return nil, fmt.Errorf("payment not found")
	}

	allowed := requesterRole == "admin" || requesterRole == "superadmin" ||
		payment.UserID.String() == requesterID
	u.log.Info("payment status access",
		zap.String("order_id", payment.OrderID),
		zap.String("requester_id", requesterID),
		zap.String("requester_role", requesterRole),
		zap.String("owner_id", payment.UserID.String()),
		zap.Bool("allowed", allowed),
	)
	if !allowed {
		return nil, domain.ErrPaymentAccessDenied
	}

	res := &responsedto.PaymentStatusResponse{
		OrderID:        payment.OrderID,
		Method:         payment.Method,
		Status:         string(payment.Status),
		MidtransStatus: payment.MidtransStatus,
		GrossAmount:    int64(math.Round(payment.GrossAmount)),
	}
	if payment.TransactionID != nil {
		res.TransactionID = payment.TransactionID.String()
	}
	if payment.PaidAt != nil {
		res.PaidAt = payment.PaidAt.Format(time.RFC3339)
	}
	return res, nil
}

// persistChargeResult saves the charged payment and, if the gateway already
// succeeded immediately (synchronous settlement), finalizes its transaction right away.
func (u *paymentUsecase) persistChargeResult(ctx context.Context, payment *domain.Payment, result *domain.GatewayChargeResult) (*responsedto.ChargePaymentResponse, error) {
	if err := u.paymentRepo.Create(ctx, payment); err != nil {
		if payment.StockReserved {
			u.releaseStock(ctx, payment.Items, payment.OrderID)
		}
		u.log.Error("failed to save payment", zap.Error(err))
		return nil, fmt.Errorf("failed to save payment")
	}

	if mapInternalStatus(result.TransactionStatus, result.FraudStatus) == domain.PaymentSuccess {
		if err := u.finalizeSuccess(ctx, payment); err != nil {
			u.log.Error("failed to finalize successful payment", zap.Error(err))
			return nil, fmt.Errorf("failed to finalize payment")
		}
		if err := u.paymentRepo.Update(ctx, payment); err != nil {
			u.log.Error("failed to update payment", zap.Error(err))
			return nil, fmt.Errorf("failed to update payment")
		}
		// Some gateway charges settle synchronously without a webhook follow-up; notify the user right away.
		u.notifyPaymentResult(ctx, payment, true)
	}

	return &responsedto.ChargePaymentResponse{
		OrderID:        payment.OrderID,
		Method:         payment.Method,
		Status:         string(payment.Status),
		GrossAmount:    int64(math.Round(payment.GrossAmount)),
		MidtransStatus: payment.MidtransStatus,
		QrString:       payment.QRString,
		QrUrl:          payment.QRURL,
		RedirectUrl:    payment.RedirectURL,
		VaNumber:       payment.VANumber,
		Bank:           payment.VABank,
		BillKey:        payment.BillKey,
		BillerCode:     payment.BillerCode,
		ExpiryTime:     result.ExpiryTime,
	}, nil
}

// finalizeSuccess creates the transaction (once, idempotent) from the cart
// stored in the payment then marks the payment successful. The transaction no_invoice =
// the payment's order_id.
func (u *paymentUsecase) finalizeSuccess(ctx context.Context, payment *domain.Payment) error {
	if payment.TransactionID != nil {
		payment.StockReserved = false // reservation already consumed by the transaction
		payment.Status = domain.PaymentSuccess
		return nil
	}

	existing, err := u.trxRepo.CheckTransactionByNoInvoice(ctx, payment.OrderID)
	if err != nil {
		return fmt.Errorf("failed to check transaction: %w", err)
	}
	if existing == nil {
		details := make([]requestdto.AddTransactionDetailRequest, 0, len(payment.Items))
		for _, it := range payment.Items {
			details = append(details, requestdto.AddTransactionDetailRequest{
				ProductId: it.ProductID.String(),
				Qty:       it.Qty,
			})
		}
		var customerID *string
		if payment.CustomerID != nil {
			s := payment.CustomerID.String()
			customerID = &s
		}
		var bank *string
		if payment.Method == "va" && payment.VABank != "" {
			b := payment.VABank
			bank = &b
		}
		addReq := &requestdto.AddTransactionRequest{
			NoInvoice:   payment.OrderID,
			TypePayment: methodToPaymentType(payment.Method),
			Bank:        bank,
			UserId:      payment.UserID.String(),
			CustomerId:  customerID,
			Details:     details,
		}
		// A reserved payment's stock was already deducted at charge time — use
		// the prepaid path so it is not deducted twice.
		addTrx := u.trxUsecase.AddTransaction
		if payment.StockReserved {
			addTrx = u.trxUsecase.AddPrepaidTransaction
		}
		if _, err := addTrx(ctx, addReq); err != nil {
			return fmt.Errorf("failed to create transaction from payment: %w", err)
		}
		existing, err = u.trxRepo.CheckTransactionByNoInvoice(ctx, payment.OrderID)
		if err != nil {
			return fmt.Errorf("failed to reload transaction: %w", err)
		}
	}

	if existing != nil {
		payment.TransactionID = &existing.ID
	}
	payment.StockReserved = false // reservation consumed by the transaction
	now := time.Now()
	payment.PaidAt = &now
	payment.Status = domain.PaymentSuccess
	return nil
}

// notifyPaymentResult sends the payment-result push notification asynchronously
// so it does not delay or fail the caller's flow — the webhook must still
// reply 200 even if FCM has issues, so errors are only logged. The request context
// dies once the response is sent, so it is detached with WithoutCancel.
func (u *paymentUsecase) notifyPaymentResult(ctx context.Context, payment *domain.Payment, success bool) {
	userID, orderID := payment.UserID.String(), payment.OrderID
	amount := int64(math.Round(payment.GrossAmount))
	notifCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
	go func() {
		defer cancel()
		if err := u.notifier.NotifyPaymentResult(notifCtx, userID, orderID, success, amount); err != nil {
			u.log.Error("failed to send payment push notification",
				zap.Error(err), zap.String("order_id", orderID))
		}
	}()
}

// buildOrder validates each product and computes gross_amount on the server
// (price × qty), not trusting the value from the client.
func (u *paymentUsecase) buildOrder(ctx context.Context, items []itemPair) (int64, []domain.PaymentItem, error) {
	if len(items) == 0 {
		return 0, nil, fmt.Errorf("items is required")
	}
	var gross float64
	paymentItems := make([]domain.PaymentItem, 0, len(items))
	for _, it := range items {
		productID, err := uuid.Parse(it.productID)
		if err != nil {
			return 0, nil, fmt.Errorf("invalid product id format")
		}
		product, err := u.productRepo.GetProduct(ctx, productID)
		if err != nil {
			u.log.Error("failed to get product", zap.Error(err))
			return 0, nil, fmt.Errorf("product %s not found", it.productID)
		}
		if product == nil {
			return 0, nil, fmt.Errorf("product %s not found", it.productID)
		}
		// Digital products need a per-line destination (see addTransaction), which
		// the online cart/PaymentItem does not carry. Selling them online would
		// capture money then fail finalization forever, so reject up front and keep
		// digital sales on the manual POS /transactions path.
		if product.ProductType.IsDigital() {
			return 0, nil, fmt.Errorf("digital products cannot be purchased via online payment")
		}
		// Early check so the user is rejected before the order is created; the
		// atomic check stays in ReserveStock (stock can change in between).
		if product.Stock < it.qty {
			return 0, nil, fmt.Errorf("insufficient stock for product %s", it.productID)
		}
		gross += product.SellingPrice * it.qty
		paymentItems = append(paymentItems, domain.PaymentItem{ProductID: productID, Qty: it.qty})
	}
	return int64(math.Round(gross)), paymentItems, nil
}

// resolveOrderID uses the no_invoice from the client if present (and unused),
// otherwise generates a new one. The order ID must be unique in Midtrans.
func (u *paymentUsecase) resolveOrderID(ctx context.Context, requested string) (string, error) {
	orderID := strings.TrimSpace(requested)
	if orderID == "" {
		orderID = generateInvoice()
	}
	existing, err := u.paymentRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		u.log.Error("failed to check existing payment", zap.Error(err))
		return "", fmt.Errorf("failed to check existing payment")
	}
	if existing != nil {
		return "", fmt.Errorf("payment with invoice %s already exists", orderID)
	}
	return orderID, nil
}

func (u *paymentUsecase) newPayment(orderID, method string, userID uuid.UUID, customerID *uuid.UUID, gross int64, items []domain.PaymentItem, result *domain.GatewayChargeResult) *domain.Payment {
	return &domain.Payment{
		OrderID:        orderID,
		UserID:         userID,
		CustomerID:     customerID,
		Method:         method,
		GrossAmount:    float64(gross),
		Status:         domain.PaymentPending,
		MidtransTrxID:  result.TransactionID,
		MidtransStatus: result.TransactionStatus,
		FraudStatus:    result.FraudStatus,
		QRString:       result.QRString,
		QRURL:          result.QRURL,
		RedirectURL:    result.RedirectURL,
		VABank:         result.Bank,
		VANumber:       result.VANumber,
		BillKey:        result.BillKey,
		BillerCode:     result.BillerCode,
		Items:          items,
		ExpiryTime:     parseMidtransTime(result.ExpiryTime),
	}
}

// releaseStock returns the reserved stock. A failure here cannot
// be retried automatically — it is flagged in the log for manual correction.
func (u *paymentUsecase) releaseStock(ctx context.Context, items []domain.PaymentItem, orderID string) {
	if err := u.productRepo.RestoreStock(ctx, items); err != nil {
		u.log.Error("PAYMENT_RECONCILIATION_REQUIRED: failed to restore reserved stock",
			zap.Error(err), zap.String("order_id", orderID))
	}
}

// --- helpers ---

type itemPair struct {
	productID string
	qty       float64
}

func toItemPairs(items []requestdto.PaymentItemRequest) []itemPair {
	out := make([]itemPair, 0, len(items))
	for _, it := range items {
		out = append(out, itemPair{productID: it.ProductId, qty: it.Qty})
	}
	return out
}

func parseOptionalUUID(s *string) (*uuid.UUID, error) {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil, nil
	}
	id, err := uuid.Parse(strings.TrimSpace(*s))
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// Midtrans standard per-transaction fees, passed on to the buyer. Percentages
// are rounded UP to the nearest rupiah so the merchant never absorbs a fraction.
// Source: Midtrans pricing page (QRIS 0.7%, Virtual Account Rp4.000 flat).
const (
	feeQrisRate = 0.007
	feeVAFlat   = 4000
)

// applyFee returns gross = subtotal + channel fee for online charges.
func applyFee(method string, subtotal int64) int64 {
	switch method {
	case "qris":
		return subtotal + int64(math.Ceil(float64(subtotal)*feeQrisRate))
	case "va":
		return subtotal + feeVAFlat
	default:
		return subtotal
	}
}

// methodToPaymentType maps the online payment method to the internal POS
// payment enum string. VA settles as a bank transfer; QRIS stays qris.
func methodToPaymentType(method string) string {
	if method == "va" {
		return "transfer"
	}
	return "qris"
}

// mapInternalStatus maps the raw Midtrans status to the internal status.
func mapInternalStatus(trxStatus, fraudStatus string) domain.PaymentStatus {
	switch strings.ToLower(trxStatus) {
	case "capture":
		if strings.EqualFold(fraudStatus, "accept") {
			return domain.PaymentSuccess
		}
		if strings.EqualFold(fraudStatus, "challenge") {
			return domain.PaymentPending
		}
		return domain.PaymentFailed
	case "settlement":
		return domain.PaymentSuccess
	case "pending":
		return domain.PaymentPending
	case "deny", "cancel":
		return domain.PaymentFailed
	case "expire":
		return domain.PaymentExpired
	default:
		return domain.PaymentPending
	}
}

func generateInvoice() string {
	return invoicePrefix + strings.ToUpper(strings.ReplaceAll(uuid.NewString()[:13], "-", ""))
}

// jakartaLoc is used to parse Midtrans times sent in WIB without an offset.
var jakartaLoc = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.FixedZone("WIB", 7*3600)
	}
	return loc
}()

// parseMidtransTime parses "2006-01-02 15:04:05" (WIB); nil if empty
// or the format is unrecognized.
func parseMidtransTime(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.ParseInLocation("2006-01-02 15:04:05", s, jakartaLoc)
	if err != nil {
		return nil
	}
	return &t
}
