package repository

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// maxTxRetries bounds how many times a write transaction is re-run on a
// transient conflict. 3 attempts clears virtually all real-world contention
// without risking a long stall.
const maxTxRetries = 3

// isRetryableTxError reports whether err is a transient Postgres transaction
// conflict worth retrying: serialization_failure (40001) or deadlock_detected
// (40P01). Everything else (including business errors and domain.ErrInternal
// wrappers) is returned as-is. errors.As walks the wrap chain, so it still
// matches when the driver error is wrapped by internalErr/fmt.Errorf.
func isRetryableTxError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "40001" || pgErr.Code == "40P01"
	}
	return false
}

// runTx runs fn (a DB transaction) and retries it on a transient
// serialization/deadlock conflict with a small randomized backoff. On success
// or a non-retryable error it returns immediately, so the normal-path behavior
// and output are unchanged — only spurious conflict failures are turned into a
// transparent re-run. fn MUST be idempotent; the transaction closures here are,
// because they re-read every row under FOR UPDATE on each attempt.
func runTx(ctx context.Context, fn func() error) error {
	var err error
	for attempt := 0; attempt < maxTxRetries; attempt++ {
		err = fn()
		if err == nil || !isRetryableTxError(err) {
			return err
		}
		// Respect cancellation/deadline instead of sleeping blindly.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(txBackoff(attempt)):
		}
	}
	return err
}

// txBackoff returns an exponential base (5ms, 10ms, 20ms) plus jitter so
// competing transactions do not retry in lockstep.
func txBackoff(attempt int) time.Duration {
	base := time.Duration(5<<attempt) * time.Millisecond
	return base + time.Duration(rand.Int63n(int64(base)+1))
}

// runTxDB runs a GORM transaction with the transient-conflict retry policy of
// runTx. Call sites swap `db.WithContext(ctx).Transaction(fn)` for
// `runTxDB(ctx, db, fn)` — the closure and its result are unchanged.
func runTxDB(ctx context.Context, db *gorm.DB, fn func(tx *gorm.DB) error) error {
	return runTx(ctx, func() error {
		return db.WithContext(ctx).Transaction(fn)
	})
}
