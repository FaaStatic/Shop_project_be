-- +goose Up
-- Tabel payments: siklus hidup pembayaran via Midtrans (QRIS & kartu). order_id
-- sama dengan no_invoice pada tabel transactions; saat pembayaran sukses,
-- transaksi dibuat dengan no_invoice = order_id ini.

-- Longgarkan check payment_type agar mendukung "kartu" (4) selain
-- tunai(0)/hutang(1)/transfer(2)/qris(3).
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS chk_transactions_payment_type;
ALTER TABLE transactions ADD CONSTRAINT chk_transactions_payment_type
    CHECK (payment_type IN (0, 1, 2, 3, 4));

CREATE TABLE IF NOT EXISTS payments (
    id                      uuid DEFAULT gen_random_uuid() NOT NULL,
    order_id                varchar(50) NOT NULL,
    user_id                 uuid NOT NULL,
    customer_id             uuid,
    method                  varchar(20) NOT NULL,
    gross_amount            numeric(15,2) NOT NULL,
    status                  varchar(20) NOT NULL DEFAULT 'pending',
    midtrans_trx_id         varchar(100),
    midtrans_status         varchar(30),
    fraud_status            varchar(30),
    qr_string               text,
    qr_url                  text,
    redirect_url            text,
    items                   jsonb,
    expiry_time             timestamptz,
    paid_at                 timestamptz,
    transaction_id          uuid,
    created_at              timestamptz,
    updated_at              timestamptz,
    deleted_at              timestamptz,
    CONSTRAINT payments_pkey PRIMARY KEY (id),
    CONSTRAINT chk_payments_status CHECK (status IN ('pending', 'success', 'failed', 'expired')),
    CONSTRAINT fk_payments_user FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT fk_payments_customer FOREIGN KEY (customer_id) REFERENCES customers (id),
    CONSTRAINT fk_payments_transaction FOREIGN KEY (transaction_id) REFERENCES transactions (id)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_order_id ON payments (order_id);
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments (status);
CREATE INDEX IF NOT EXISTS idx_payments_deleted_at ON payments (deleted_at);

-- +goose Down
DROP TABLE IF EXISTS payments;
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS chk_transactions_payment_type;
ALTER TABLE transactions ADD CONSTRAINT chk_transactions_payment_type
    CHECK (payment_type IN (0, 1, 2, 3));
