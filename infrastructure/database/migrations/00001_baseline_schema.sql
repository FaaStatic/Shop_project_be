-- +goose Up
-- Baseline schema. Mencerminkan kondisi tabel saat migrasi versioned (goose)
-- mulai diadopsi. Sengaja idempoten (CREATE ... IF NOT EXISTS + constraint
-- inline) supaya aman dijalankan baik pada database yang SUDAH berisi tabel
-- (no-op, data lama tetap utuh) maupun database baru (membuat dari nol).
-- Perubahan skema selanjutnya WAJIB lewat file migration baru, bukan AutoMigrate.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users (
    id         uuid DEFAULT gen_random_uuid() NOT NULL,
    username   varchar(100) NOT NULL,
    password   varchar(255) NOT NULL,
    role       smallint DEFAULT 2,
    created_at timestamptz,
    updated_at timestamptz,
    deleted_at timestamptz,
    CONSTRAINT users_pkey PRIMARY KEY (id),
    CONSTRAINT chk_users_role CHECK (role IN (0, 1, 2))
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users (username);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users (deleted_at);

CREATE TABLE IF NOT EXISTS customers (
    id         uuid DEFAULT gen_random_uuid() NOT NULL,
    name       varchar(150) NOT NULL,
    phone      varchar(15),
    address    text,
    created_at timestamptz,
    updated_at timestamptz,
    deleted_at timestamptz,
    CONSTRAINT customers_pkey PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_customers_deleted_at ON customers (deleted_at);

CREATE TABLE IF NOT EXISTS products (
    id                 uuid DEFAULT gen_random_uuid() NOT NULL,
    sku                varchar(50),
    product_name       varchar(255) NOT NULL,
    unit               smallint NOT NULL,
    purchase_price     numeric(15,2) NOT NULL,
    selling_price      numeric(15,2) NOT NULL,
    selling_price_debt numeric(15,2) NOT NULL,
    stock              numeric(10,2) DEFAULT 0,
    category           varchar(100),
    image              text,
    created_at         timestamptz,
    updated_at         timestamptz,
    deleted_at         timestamptz,
    CONSTRAINT products_pkey PRIMARY KEY (id),
    CONSTRAINT chk_products_unit CHECK (unit IN (0, 1, 2, 3, 4, 5))
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_products_sku ON products (sku);
CREATE INDEX IF NOT EXISTS idx_products_category ON products (category);
CREATE INDEX IF NOT EXISTS idx_products_deleted_at ON products (deleted_at);

CREATE TABLE IF NOT EXISTS debts (
    id             uuid DEFAULT gen_random_uuid() NOT NULL,
    customer_id    uuid NOT NULL,
    total_debt     numeric(15,2) NOT NULL,
    remaining_debt numeric(15,2) NOT NULL,
    status         smallint DEFAULT 0,
    due_date       date,
    created_at     timestamptz,
    updated_at     timestamptz,
    deleted_at     timestamptz,
    CONSTRAINT debts_pkey PRIMARY KEY (id),
    CONSTRAINT chk_debts_status CHECK (status IN (0, 1)),
    CONSTRAINT fk_customers_debts FOREIGN KEY (customer_id) REFERENCES customers (id)
);
CREATE INDEX IF NOT EXISTS idx_debts_deleted_at ON debts (deleted_at);

CREATE TABLE IF NOT EXISTS debt_payments (
    id            uuid DEFAULT gen_random_uuid() NOT NULL,
    debt_id       uuid NOT NULL,
    user_id       uuid NOT NULL,
    nominal_bayar numeric(15,2) NOT NULL,
    tanggal_bayar timestamptz,
    CONSTRAINT debt_payments_pkey PRIMARY KEY (id),
    CONSTRAINT fk_debts_debt_payments FOREIGN KEY (debt_id) REFERENCES debts (id),
    CONSTRAINT fk_debt_payments_user FOREIGN KEY (user_id) REFERENCES users (id)
);

CREATE TABLE IF NOT EXISTS transactions (
    id                uuid DEFAULT gen_random_uuid() NOT NULL,
    no_invoice        varchar(50) NOT NULL,
    user_id           uuid NOT NULL,
    customer_id       uuid,
    debt_id           uuid,
    payment_type      smallint NOT NULL,
    total_transaction numeric(15,2) NOT NULL,
    created_at        timestamptz,
    updated_at        timestamptz,
    deleted_at        timestamptz,
    CONSTRAINT transactions_pkey PRIMARY KEY (id),
    CONSTRAINT chk_transactions_payment_type CHECK (payment_type IN (0, 1, 2, 3)),
    CONSTRAINT fk_customers_transactions FOREIGN KEY (customer_id) REFERENCES customers (id),
    CONSTRAINT fk_debts_transactions FOREIGN KEY (debt_id) REFERENCES debts (id),
    CONSTRAINT fk_users_transactions FOREIGN KEY (user_id) REFERENCES users (id)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_transactions_no_invoice ON transactions (no_invoice);
CREATE INDEX IF NOT EXISTS idx_transactions_debt_id ON transactions (debt_id);
CREATE INDEX IF NOT EXISTS idx_transactions_deleted_at ON transactions (deleted_at);

CREATE TABLE IF NOT EXISTS transactions_detail (
    id             uuid DEFAULT gen_random_uuid() NOT NULL,
    transaction_id uuid NOT NULL,
    product_id     uuid NOT NULL,
    price          numeric(15,2) NOT NULL,
    price_debt     numeric(15,2) NOT NULL,
    qty            numeric(8,2) NOT NULL,
    subtotal       numeric(15,2) NOT NULL,
    CONSTRAINT transactions_detail_pkey PRIMARY KEY (id),
    CONSTRAINT fk_transactions_transaction_detail FOREIGN KEY (transaction_id) REFERENCES transactions (id),
    CONSTRAINT fk_transactions_detail_product FOREIGN KEY (product_id) REFERENCES products (id)
);

-- +goose Down
DROP TABLE IF EXISTS transactions_detail;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS debt_payments;
DROP TABLE IF EXISTS debts;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS customers;
DROP TABLE IF EXISTS users;
