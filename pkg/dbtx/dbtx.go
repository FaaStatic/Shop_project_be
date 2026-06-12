// Package dbtx menyediakan "transaction manager" sederhana: cara menjalankan
// beberapa operasi repository di dalam SATU transaksi database tanpa membuat
// repository saling tahu satu sama lain.
//
// Idenya: Manager.Do membuka satu transaksi GORM lalu menitipkan objek *gorm.DB
// transaksi tersebut ke dalam context. Setiap repository memanggil Conn(ctx, db)
// untuk mengambil koneksi yang tepat — kalau sedang berada di dalam Do, ia
// memakai transaksi yang sama; kalau tidak, ia memakai koneksi biasa. Dengan
// begitu repo yang sudah ada tetap dipakai apa adanya, tetapi bisa dijalankan
// secara atomik saat dibutuhkan.
package dbtx

import (
	"context"

	"gorm.io/gorm"
)

// txKey adalah kunci privat untuk menyimpan *gorm.DB transaksi di context.
type txKey struct{}

// WithTx mengembalikan turunan ctx yang membawa transaksi tx.
func WithTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// Conn mengembalikan koneksi yang harus dipakai repository: transaksi dari ctx
// bila ada (sedang di dalam Manager.Do), selain itu fallback (db biasa).
func Conn(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok && tx != nil {
		return tx
	}
	return fallback
}

// Manager menjalankan sekumpulan operasi dalam satu transaksi database.
type Manager struct {
	db *gorm.DB
}

// NewManager membuat Manager dari koneksi database utama.
func NewManager(db *gorm.DB) *Manager {
	return &Manager{db: db}
}

// Do menjalankan fn di dalam satu transaksi. Bila fn mengembalikan error,
// seluruh perubahan di-rollback; bila nil, di-commit. Jika ctx sudah berada di
// dalam transaksi lain, transaksi itu dipakai ulang (tidak membuka yang baru)
// agar tidak terjadi nested-begin yang tidak diinginkan.
func (m *Manager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok && tx != nil {
		return fn(ctx)
	}
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(WithTx(ctx, tx))
	})
}
