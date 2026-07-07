package domain

import "errors"

// ErrDuplicateInvoice is returned when saving a transaction with a no_invoice
// that is already in use. Detected from the database unique constraint (not a
// manual check), so it is safe against a race between two concurrent requests.
var ErrDuplicateInvoice = errors.New("transaction with this invoice already exists")

// ErrInternal marks an infrastructure/DB failure whose raw cause must not reach
// the client (it may embed driver/schema detail). Repositories wrap such errors
// with it so the usecase layer can log the full cause server-side yet return a
// clean generic message. Business/validation errors (not found, insufficient
// stock, already exists) are returned plain so their message stays visible.
var ErrInternal = errors.New("internal server error")

// ErrInvalidSignature is returned by the payment webhook flow when the Midtrans
// signature_key does not match. Handlers map it to HTTP 403 (do not retry)
// instead of comparing error strings.
var ErrInvalidSignature = errors.New("invalid signature")
