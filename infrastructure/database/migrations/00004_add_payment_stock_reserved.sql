-- +goose Up
-- stock_reserved: true selama stok item masih tereservasi untuk pembayaran ini.
-- Stok dipotong saat charge dibuat (mencegah oversell), dikembalikan saat
-- pembayaran hangus (failed/expired), dan dikonsumsi saat pembayaran sukses.
ALTER TABLE payments ADD COLUMN IF NOT EXISTS stock_reserved boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE payments DROP COLUMN IF EXISTS stock_reserved;
