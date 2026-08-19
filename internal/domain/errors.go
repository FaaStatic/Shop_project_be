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

// ErrNotFound marks a "no such record" condition. Repositories return it via
// the NotFound constructor so usecases and handlers can detect it with
// errors.Is and map it to HTTP 404 without string matching.
var ErrNotFound = errors.New("not found")

// ErrInvalidID marks a malformed identifier (e.g. a non-UUID) supplied by the
// client. Usecases return it via the InvalidID constructor so handlers map it
// to HTTP 400.
var ErrInvalidID = errors.New("invalid id")

// notFoundError keeps a descriptive message while matching ErrNotFound.
type notFoundError struct{ msg string }

func (e *notFoundError) Error() string        { return e.msg }
func (e *notFoundError) Is(target error) bool { return target == ErrNotFound }

// NotFound builds a descriptive not-found error that errors.Is(err, ErrNotFound)
// detects.
func NotFound(msg string) error { return &notFoundError{msg: msg} }

// invalidIDError keeps a descriptive message while matching ErrInvalidID.
type invalidIDError struct{ msg string }

func (e *invalidIDError) Error() string        { return e.msg }
func (e *invalidIDError) Is(target error) bool { return target == ErrInvalidID }

// InvalidID builds a descriptive invalid-id error that errors.Is(err, ErrInvalidID)
// detects.
func InvalidID(msg string) error { return &invalidIDError{msg: msg} }

// duplicateError keeps a descriptive message while matching ErrDuplicateInvoice.
type duplicateError struct{ msg string }

func (e *duplicateError) Error() string        { return e.msg }
func (e *duplicateError) Is(target error) bool { return target == ErrDuplicateInvoice }

// Duplicate builds a descriptive duplicate-key error that errors.Is(err, ErrDuplicateInvoice)
// detects so handlers can map it to HTTP 409.
func Duplicate(msg string) error { return &duplicateError{msg: msg} }
