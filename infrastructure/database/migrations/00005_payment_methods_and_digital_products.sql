-- +goose Up
-- Metode pembayaran (Cash/QRIS/VA BCA+Mandiri) & produk digital.

-- Produk digital: 0=physical (default), 1=digital (pulsa/e-wallet/data).
ALTER TABLE products ADD COLUMN IF NOT EXISTS product_type smallint NOT NULL DEFAULT 0;
ALTER TABLE products DROP CONSTRAINT IF EXISTS products_product_type_check;
ALTER TABLE products ADD CONSTRAINT products_product_type_check CHECK (product_type IN (0,1));

-- Bank VA pada transaksi (diisi hanya saat payment_type = transfer).
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS bank varchar(20) NULL;

-- Nomor tujuan (HP/akun e-wallet) untuk baris produk digital.
ALTER TABLE transactions_detail ADD COLUMN IF NOT EXISTS destination varchar(50) NULL;

-- Ketatkan payment_type: kartu (4) dihapus. Guard: hanya jika tidak ada baris = 4.
DO $$
BEGIN
	IF NOT EXISTS (SELECT 1 FROM transactions WHERE payment_type = 4) THEN
		ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_payment_type_check;
		ALTER TABLE transactions ADD CONSTRAINT transactions_payment_type_check CHECK (payment_type IN (0,1,2,3));
	END IF;
END $$;

-- Kolom VA pada payments.
ALTER TABLE payments ADD COLUMN IF NOT EXISTS va_bank varchar(20) NULL;
ALTER TABLE payments ADD COLUMN IF NOT EXISTS va_number varchar(50) NULL;
ALTER TABLE payments ADD COLUMN IF NOT EXISTS bill_key varchar(50) NULL;
ALTER TABLE payments ADD COLUMN IF NOT EXISTS biller_code varchar(20) NULL;

-- +goose Down
ALTER TABLE payments DROP COLUMN IF EXISTS biller_code;
ALTER TABLE payments DROP COLUMN IF EXISTS bill_key;
ALTER TABLE payments DROP COLUMN IF EXISTS va_number;
ALTER TABLE payments DROP COLUMN IF EXISTS va_bank;
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_payment_type_check;
ALTER TABLE transactions ADD CONSTRAINT transactions_payment_type_check CHECK (payment_type IN (0,1,2,3,4));
ALTER TABLE transactions_detail DROP COLUMN IF EXISTS destination;
ALTER TABLE transactions DROP COLUMN IF EXISTS bank;
ALTER TABLE products DROP CONSTRAINT IF EXISTS products_product_type_check;
ALTER TABLE products DROP COLUMN IF EXISTS product_type;
