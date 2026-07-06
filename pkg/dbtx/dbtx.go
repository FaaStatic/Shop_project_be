// Package dbtx provides a simple "transaction manager": a way to run
// several repository operations within ONE database transaction without making
// the repositories aware of each other.
//
// The idea: Manager.Do opens one GORM transaction then stashes that *gorm.DB
// transaction object into the context. Each repository calls Conn(ctx, db)
// to obtain the right connection — if inside Do, it
// uses the same transaction; otherwise it uses the normal connection. This
// way existing repos are used as-is, yet can be run
// atomically when needed.
package dbtx

import (
	"context"

	"gorm.io/gorm"
)

// txKey is the private key for storing the *gorm.DB transaction in the context.
type txKey struct{}

// WithTx returns a derived ctx carrying the transaction tx.
func WithTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// Conn returns the connection a repository should use: the transaction from ctx
// if present (inside Manager.Do), otherwise the fallback (normal db).
func Conn(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok && tx != nil {
		return tx
	}
	return fallback
}

// Manager runs a set of operations within a single database transaction.
type Manager struct {
	db *gorm.DB
}

// NewManager builds a Manager from the main database connection.
func NewManager(db *gorm.DB) *Manager {
	return &Manager{db: db}
}

// Do runs fn within one transaction. If fn returns an error,
// all changes are rolled back; if nil, committed. If ctx is already inside
// another transaction, that transaction is reused (no new one is opened)
// to avoid an unwanted nested-begin.
func (m *Manager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok && tx != nil {
		return fn(ctx)
	}
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(WithTx(ctx, tx))
	})
}
