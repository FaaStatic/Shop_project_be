-- +goose Up
-- Tabel device_tokens: token FCM per perangkat untuk push notification.
-- user_id nullable — saat logout token dilepas dari user (bukan dihapus)
-- agar perangkat tetap terdaftar dan bisa dipakai login berikutnya.
CREATE TABLE IF NOT EXISTS device_tokens (
    id           uuid DEFAULT gen_random_uuid() NOT NULL,
    token        text NOT NULL,
    user_id      uuid,
    device_id    text NOT NULL,
    platform     varchar(16) NOT NULL,
    created_at   timestamptz,
    updated_at   timestamptz,
    last_used_at timestamptz,
    CONSTRAINT device_tokens_pkey PRIMARY KEY (id),
    CONSTRAINT fk_device_tokens_user FOREIGN KEY (user_id) REFERENCES users (id)
);
-- Unique index pada token diperlukan oleh upsert ON CONFLICT (token) di repository.
CREATE UNIQUE INDEX IF NOT EXISTS idx_device_tokens_token ON device_tokens (token);
CREATE INDEX IF NOT EXISTS idx_device_tokens_user_id ON device_tokens (user_id);

-- +goose Down
DROP TABLE IF EXISTS device_tokens;
