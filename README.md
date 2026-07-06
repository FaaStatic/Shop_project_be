# Shop POS Backend — Go Fiber v3

Backend point-of-sale (POS) untuk toko/warung kecil, dibangun dengan **Go Fiber v3** dan Clean Architecture. Menangani produk (fisik & digital), pelanggan, transaksi penjualan, hutang (kasbon), pembayaran online lewat Midtrans (QRIS & Virtual Account), notifikasi push (FCM), serta autentikasi kasir dengan sesi berbasis Redis. Laporan penjualan dan hutang bisa diekspor ke Excel dan PDF.

## Tujuan

Menyediakan API kasir yang ringan namun aman untuk toko kecil: pencatatan stok, transaksi penjualan yang atomik (stok dan hutang berubah dalam satu transaksi DB), pelacakan hutang pelanggan, pembayaran online, dan laporan bulanan. Akses dibedakan antara **kasir/staff** dan **superadmin** lewat JWT + role.

## Fitur

- **Modul domain**: produk, pelanggan, transaksi, hutang (kasbon), pembayaran, FCM, user/sesi
- **Produk fisik & digital**: `ProductType` membedakan produk fisik (dikelola stok) dari produk digital (pulsa/e-wallet/paket data — tanpa stok, butuh nomor tujuan saat penjualan). Produk digital dilewati dari pengurangan/pengembalian stok
- **Transaksi penjualan atomik**: penyesuaian stok dan pencatatan hutang dilakukan dalam satu transaksi database (lihat [pkg/dbtx/](pkg/dbtx/))
- **Pembayaran online (Midtrans)**: charge **QRIS** dan **Virtual Account** (BCA `bank_transfer`, Mandiri `echannel`); harga item dihitung server-side (tidak dikirim client) dan biaya admin ditambahkan otomatis (QRIS 0.7%, VA flat). Status pembayaran diperbarui lewat webhook yang diverifikasi `signature_key`. Produk digital **hanya lewat POS**, ditolak pada charge online
- **Notifikasi push (FCM)**: registrasi & logout device token via Firebase Cloud Messaging
- **Autentikasi & otorisasi**: login/register staff, access + refresh token JWT, sesi disimpan di Redis, role-based access (`superadmin` vs `staff`)
- **Pembuatan akun privileged via CLI**: akun `superadmin` dibuat lewat command `create-admin`, bukan endpoint publik
- **Laporan**: ekspor Excel (excelize) dan PDF (fpdf) untuk struk, laporan bulanan, dan rekap hutang; file PDF disajikan sebagai download di `/storage/reports`
- **Keamanan bawaan**: rate limiting global + per-endpoint login & webhook (storage Redis), Helmet/XSS headers, CORS, CSRF, encrypt-cookie (produksi), trusted proxy untuk pembacaan IP asli
- **Migrasi versioned dengan goose**: file SQL di-embed ke binary, dijalankan lewat command `migrate` / `migrate-reset`
- **Konfigurasi via Viper** dari file YAML per-environment (`.config.development.yaml` / `.config.production.yaml`)
- **CLI dengan Cobra**: `serve`, `migrate`, `migrate-reset`, `create-admin`
- **Swagger UI** disajikan lewat `gofiber/contrib/v3/swaggerui` di root (`/`)
- Validasi request dengan go-playground/validator, JSON via bytedance/sonic

## Arsitektur

```
├── main.go              # Entry point — memanggil cmd.Execute()
├── cmd/                 # Cobra commands (serve, migrate, migrate-reset, create-admin)
├── config/
│   ├── env_config/      # Loader & validasi config (Viper)
│   └── fiber_config/    # Setup Fiber, middleware chain, Swagger
├── infrastructure/
│   ├── database/        # Init Postgres (GORM) + migrasi goose (SQL embed)
│   ├── cache/           # Redis (sesi + storage rate limiter)
│   ├── api/payment/     # Gateway Midtrans (charge QRIS/VA, verifikasi webhook)
│   ├── fcm/             # Sender Firebase Cloud Messaging
│   └── logger/          # Logger zap kustom
├── internal/
│   ├── domain/          # Entitas & interface (model, errors, tx)
│   ├── dto/             # Request/response DTO per domain
│   ├── repository/      # Akses data (GORM / Redis)
│   ├── usecase/         # Business logic
│   ├── delivery/http/
│   │   ├── handler/     # Fiber v3 handlers (Ctx sebagai value type)
│   │   ├── middleware/  # JWT, CORS, CSRF, rate limit, XSS, compress, dll.
│   │   └── route/       # Pendaftaran route
│   └── constant/        # Enum (role, payment, product type, status hutang) & paginasi
└── pkg/
    ├── jwt/             # Generate/verifikasi token
    ├── dbtx/            # Helper transaksi DB
    ├── validator/       # StructValidator Fiber
    ├── response/        # Format response standar
    ├── pdf/             # Laporan PDF (struk, hutang, bulanan)
    └── sheet/           # Ekspor Excel
```

### Alur request

`route` → `middleware` (JWT + role) → `handler` → `usecase` → `repository` → Postgres/Redis. Endpoint `/auth/*` dan webhook `/payments/notification` bersifat publik (webhook diverifikasi via `signature_key`, bukan JWT); sisanya berada di bawah grup `/api` yang dilindungi JWT, dengan beberapa aksi sensitif (delete, update produk, laporan bulanan & hutang) dibatasi hanya untuk `superadmin`.

## Endpoint utama

| Grup | Endpoint | Akses |
|------|----------|-------|
| Auth | `POST /auth/login`, `POST /auth/register` | Publik (rate-limited; register hanya staff) |
| Payments | `POST /payments/notification` (webhook Midtrans) | Publik (rate-limited; diverifikasi `signature_key`) |
| Payments | `POST /api/payments/qris`, `POST /api/payments/va`, `GET /api/payments/:order_id/status` | JWT |
| Products | `POST /api/products`, `POST /api/products/bulk`, `GET /api/products`, `GET /api/products/:id`, `PATCH /api/products/stock` | JWT |
| Products | `PUT /api/products`, `DELETE /api/products` | superadmin |
| Transactions | `POST /api/transactions`, `GET /api/transactions`, `GET /api/transactions/:id`, `GET /api/transactions/report/transaction` | JWT |
| Transactions | `GET /api/transactions/report/month`, `DELETE /api/transactions` | superadmin |
| Customers | `POST /api/customers`, `GET /api/customers`, `GET /api/customers/:id`, `PUT /api/customers` | JWT |
| Customers | `DELETE /api/customers` | superadmin |
| Debts | `POST /api/debts`, `GET /api/debts`, `GET /api/debts/:id` | JWT |
| Debts | `GET /api/debts/report`, `DELETE /api/debts` | superadmin |
| FCM | `POST /api/fcm/register`, `POST /api/fcm/logout` | JWT |

Dokumentasi lengkap tersedia di Swagger UI (root `/`) setelah server berjalan.

## Stack

Go 1.26 · Fiber v3 · GORM + PostgreSQL · goose (migrasi) · Redis · Midtrans · Firebase FCM · Cobra · Viper · zap · Swagger UI · excelize · fpdf · go-playground/validator · bytedance/sonic

## Menjalankan

### 1. Prasyarat

- Go 1.26+
- PostgreSQL
- Redis

### 2. Konfigurasi

Salin `.config.example.yaml` menjadi `.config.development.yaml` (atau `.config.production.yaml`) dan isi nilainya:

```bash
cp .config.example.yaml .config.development.yaml
```

```yaml
server:
  port: 3030
  name: novi_shop
  env: development
  host: localhost
  trusted_proxies: []      # isi IP/CIDR reverse proxy bila di belakang nginx
database:
  user: postgres
  pass: secret
  port: 5432
  dbname: shop
  host: localhost
  sslmode: disable
  time_zone: Asia/Jakarta
redis:
  url:
  db: 0
  username:
  password:
  port: 6379
  host: localhost
jwt:
  secret: ganti-dengan-secret-kuat
  token_ttl: 3600          # detik
  refresh_token_ttl: 86400 # detik
encrypt:
  key: 32-karakter-key-untuk-encrypt-cookie
midtrans:
  server_key: SB-Mid-server-xxxx
  client_key: SB-Mid-client-xxxx
  environment: sandbox     # sandbox | production
firebase:
  google_application_credentials: ./serviceAccountKey.json  # path kredensial FCM
```

File config dipilih berdasarkan variabel lingkungan `APP_ENV` (`development` atau `production`). Nilai juga bisa dioverride lewat env var (Viper `AutomaticEnv`). Field `midtrans.*` dan `firebase.google_application_credentials` wajib diisi (divalidasi saat startup).

### 3. Migrasi database

```bash
make migrate-dev        # APP_ENV=development go run main.go migrate
# atau reset (drop & migrate ulang):
make migrate-reset-dev
```

### 4. Buat akun superadmin

Endpoint register publik hanya membuat akun `staff`. Akun `superadmin` dibuat lewat CLI:

```bash
APP_ENV=development go run main.go create-admin \
  --username admin --password 'passwordKuat' --role superadmin
```

### 5. Jalankan server

```bash
make serve-dev          # APP_ENV=development go run main.go serve
# produksi:
make serve              # APP_ENV=production go run main.go serve
```

Server berjalan di port sesuai `server.port` (default contoh: `3030`). Swagger UI tersedia di root `/`.

### Perintah Makefile lain

```bash
make build / build-dev  # build binary
make swagger            # generate swagger.json dari anotasi (butuh swag)
make test               # APP_ENV=development go test -v ./...
make fmt / tidy / clean
```

## Catatan Fiber v3

Project ini menargetkan Fiber **v3** (bukan v2): handler menerima `fiber.Ctx` sebagai value type, dan paket contrib memakai module path `/v3` — perlu diperhatikan karena sebagian besar contoh online masih memakai idiom v2.
