package domain

import "context"

// TxManager menjalankan beberapa operasi repository di dalam satu transaksi
// database (atomik). Implementasinya ada di pkg/dbtx. Usecase memakai interface
// ini agar tidak bergantung langsung pada paket infrastruktur dan mudah di-mock
// saat testing.
type TxManager interface {
	// Do menjalankan fn dalam satu transaksi: error -> rollback, nil -> commit.
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}
