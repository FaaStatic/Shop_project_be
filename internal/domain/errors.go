package domain

import "errors"

// ErrDuplicateInvoice dikembalikan saat menyimpan transaksi dengan no_invoice
// yang sudah dipakai. Dideteksi dari unique constraint database (bukan dari
// pengecekan manual), sehingga aman terhadap race dua request bersamaan.
var ErrDuplicateInvoice = errors.New("transaction with this invoice already exists")
