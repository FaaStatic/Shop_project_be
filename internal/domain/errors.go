package domain

import "errors"

// ErrDuplicateInvoice is returned when saving a transaction with a no_invoice
// that is already in use. Detected from the database unique constraint (not a
// manual check), so it is safe against a race between two concurrent requests.
var ErrDuplicateInvoice = errors.New("transaction with this invoice already exists")
