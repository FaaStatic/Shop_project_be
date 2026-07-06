# Design: Metode Pembayaran (Cash / QRIS / VA BCA+Mandiri) & Produk Digital

**Tanggal:** 2026-07-06
**Status:** Draft — menunggu review
**Catatan:** Versi ini menggantikan draft sebelumnya (yang memuat GoPay & bank BNI/BRI/Permata).

## Ringkasan

Dua perubahan yang saling terkait:

1. **Metode pembayaran** disederhanakan menjadi **Cash, QRIS, Transfer VA (BCA & Mandiri)**,
   dengan **Hutang (debt) dipertahankan**. Kartu debit/kredit dan GoPay (channel terpisah) dihapus.
   Channel online Midtrans = **QRIS + VA(BCA, Mandiri)** saja.
2. **Produk digital** baru: pengisian saldo e-wallet (DANA/GoPay/ShopeePay), pulsa, dan
   paket internet — dimodelkan sebagai produk katalog bertipe *digital* dengan **nomor tujuan**
   per baris transaksi; fulfillment manual (tanpa integrasi biller). Produk digital **tidak**
   mengurangi stok fisik.

## Keputusan yang Sudah Dikunci (hasil klarifikasi)

| Topik | Keputusan |
|---|---|
| Enum `MoneyPayment` | Pertahankan nomor lama: `tunai(0)`, `hutang(1)`, `transfer(2)`, `qris(3)`. **Buang `kartu(4)`.** `transfer` dipakai untuk VA. |
| Bank VA di transaksi | Kolom `bank` terpisah di `transactions` (`bca` / `mandiri`), diisi hanya saat `payment_type = transfer`. |
| Produk digital | Katalog + nomor tujuan, fulfillment manual. Tidak potong stok. |
| Channel online Midtrans | QRIS + VA(BCA, Mandiri). GoPay dibuang. |

## Bagian 1 — Metode Pembayaran

### 1.1 Enum `internal/constant/enum/type_payment.go`

- Hapus konstanta & branch `kartu`. Sisakan `tunai, hutang, transfer, qris`.
- `ParseMoneyPayment` & `String()` tidak lagi menerima/mengembalikan `"kartu"`.
- **Laporan aman:** angka lama dipertahankan → SQL `payment_type = 1` (hutang) di
  `GetMonthlyReport`/`GetDailyReport` tetap benar; tidak perlu migrasi data.

### 1.2 Kolom `bank` pada transaksi (`internal/domain/transactions.go`)

- Tambah `Bank *string` (nullable) di `Transactions` — `bca`|`mandiri`, hanya untuk `transfer`.
- Tambah field ke DTO: `AddTransactionRequest.Bank *string validate:"omitempty,oneof=bca mandiri"`.
- Validasi usecase: jika `type_payment == transfer` maka `bank` wajib & harus `bca|mandiri`;
  jika bukan transfer, `bank` diabaikan/di-null-kan.
- Update `oneof` di DTO: `type_payment` → `oneof=tunai hutang transfer qris` (buang `kartu`);
  `FilterTransactionRequest.TypePayment` → `oneof=0 1 2 3`.

### 1.3 Channel online Midtrans (revisi payment gateway/usecase)

Mengacu ke file payment yang sudah ada (`domain/payment.go`, `usecase/payment_usecase.go`,
`repository/payment_repository.go`, `infrastructure/api/payment/midtrans_gateway.go`,
DTO & handler/route payment):

- **Hapus** seluruh alur kartu: `ChargeCard` (usecase + gateway + interface), `ChargeCardRequest`,
  handler & route `/payments/card`.
- **Tambah** `ChargeVA` di port `PaymentGateway` (hapus `ChargeCard`). `ChargeQris` tetap.
  GoPay **tidak** ditambahkan.
- `GatewayChargeInput`: buang `CardTokenID`/`Authentication`; tambah `Bank string` (VA).
- `GatewayChargeResult`: tambah `VANumber`, `Bank`, `BillKey`, `BillerCode`.
- Adapter `ChargeVA`:
  - `bca` → `PaymentType: bank_transfer`, `BankTransfer{Bank: bca}` → ambil `va_numbers[0]`.
  - `mandiri` → `PaymentType: echannel`, `EChannel{BillInfo1, BillInfo2}` → ambil
    `bill_key` + `biller_code` (Mandiri tidak memakai `va_number` biasa).
  - Bank tak dikenal → error (defense in depth walau sudah divalidasi DTO).
- `Payment` model: `Method` = `"qris" | "va"`; tambah `VABank`, `VANumber`, `BillKey`, `BillerCode`.
- Endpoint: `/payments/qris` (tetap), tambah `/payments/va` (body `{bank: bca|mandiri, items}`).
- `methodToPaymentType`: `qris→qris`, `va→transfer` (POS enum). Kartu dihapus.
- `finalizeSuccess` mengisi `AddTransactionRequest.Bank` dari `Payment.VABank` untuk VA.

### 1.4 Fee (biaya tambahan ke pembeli) — tetap dari draft awal

Hardcoded konstanta, ditambahkan ke `gross_amount`, persentase dibulatkan **ke atas** (`math.Ceil`):

| Metode | Fee |
|---|---|
| QRIS | 0,7% |
| Virtual Account (BCA & Mandiri) | Rp4.000 flat/transaksi |

Fungsi `applyFee(method string, subtotal int64) int64` di usecase, dihitung server-side setelah
`buildOrder`. (Cash/hutang di jalur POS tidak kena fee — fee hanya untuk charge online.)

## Bagian 2 — Produk Digital (e-wallet / pulsa / paket internet)

### 2.1 Model `Products` (`internal/domain/product.go`)

- Tambah `ProductType enum.ProductType` (`physical(0)` default, `digital(1)`), kolom `smallint`
  dengan check `IN (0,1)`. Sub-jenis (DANA/GoPay/ShopeePay/pulsa/data) memakai field `Category`
  string yang sudah ada — tidak menambah enum baru (YAGNI).
- Produk digital: `Stock` diabaikan (boleh 0, dianggap unlimited), tidak direservasi/dipotong.
- Enum baru `internal/constant/enum/product_type.go` dengan `Parse`/`String`, pola sama seperti
  `MoneyPayment`/`ProductUnit`.

### 2.2 Nomor tujuan pada detail transaksi (`internal/domain/transactions.go`)

- `TransactionsDetail`: tambah `Destination *string` (nullable) — nomor HP / akun e-wallet.
- DTO `AddTransactionDetailRequest`: tambah `Destination *string`.
- Validasi usecase: jika produk `digital` → `Destination` wajib (non-kosong); jika `physical` → diabaikan.

### 2.3 Logika stok sadar-tipe (repo + usecase)

- `transaction_usecase.addTransaction`:
  - Saat loop detail: ambil produk; jika `digital`, lewati cek stok, set `Destination`.
  - Harga: produk digital pakai `SellingPrice` (SellingPriceDebt boleh = SellingPrice).
- `transaction_repository.CreateTransaction`:
  - Di blok deduksi stok, **lewati** produk `digital` (`if product.ProductType == digital { continue }`)
    sebelum cek/`Update` stok. Lock tetap aman (locking baris digital = no-op).
- `DeleteTransaction`: saat restore stok, **lewati** produk digital (tidak menambah stok balik).
- Jalur online (QRIS/VA): `ProductRepository.ReserveStock`/`RestoreStock` melewati produk
  digital. **Keputusan (final review):** produk digital **tidak** boleh dibeli via pembayaran
  online — `payment_usecase.buildOrder` **menolak** cart yang memuat item digital dengan error
  400 yang jelas. Alasan: alur online/`PaymentItem` tidak punya field `destination`, sedangkan
  `addTransaction` mewajibkannya untuk produk digital; membiarkannya lolos akan membuat
  pembayaran ter-*capture* tapi gagal difinalisasi (stuck). Penjualan produk digital hanya
  lewat POS `/transactions` (cash/transfer/hutang). Pembelian digital via QRIS/VA bisa jadi
  fitur terpisah nanti (perlu menambah `destination` ke DTO online + alur webhook).

### 2.4 DTO produk (`internal/dto/request_dto/product_request.go`)

- `AddProduct`/`UpdateProduct`: tambah `ProductType *int validate:"omitempty,oneof=0 1"`.
  Default `physical` bila kosong.

### 2.5 Bulk import produk (sadar `product_type`)

Perubahan pada `product.go` (field `ProductType`) diikutkan ke seluruh jalur bulk import:

- `pkg/sheet/product_sheet.go`: tambah field `ProductType string` (raw) di `ProductRow`,
  dibaca via `get("product_type")`. Kolom **opsional** — file lama tanpa kolom ini tetap valid
  dan default ke `physical`.
- `product_usecase.AddBulkProductShopWithLock`: setelah dedupe SKU, parse `row.ProductType`
  (kosong → `physical`; nilai tak dikenal → tambahkan ke `rowErrors` seperti unit invalid, baris
  dilewati) dan set `ProductType` pada `domain.Products` yang dibangun.
- `product_repository.AddBulkProduct`: kolom `product_type` ikut ter-insert lewat GORM (tidak ada
  perubahan query karena `CreateInBatches` menuliskan seluruh field non-zero; `physical(0)` adalah
  default kolom sehingga aman). Urutan lock/`ON CONFLICT (sku) DO NOTHING` tetap sama —
  tidak ada perubahan perilaku konkurensi.
- Produk digital yang di-import: `Stock` boleh 0 (diabaikan saat transaksi). Tidak ada perubahan
  pada logika anti-deadlock (sort by SKU) maupun batasan `batchSize`.

## Bagian 3 — Migrasi Database (goose)

File baru mengikuti konvensi `infrastructure/database/migrations/` (terbaru `00004_...`):

- `00005_payment_methods_and_digital_products.sql`:
  - `ALTER TABLE transactions ADD COLUMN bank varchar(20) NULL;`
  - `ALTER TABLE transactions_detail ADD COLUMN destination varchar(50) NULL;`
  - `ALTER TABLE products ADD COLUMN product_type smallint NOT NULL DEFAULT 0;`
  - Update check constraint `payment_type`: dari `IN (0,1,2,3,4)` → `IN (0,1,2,3)`.
    **Guard:** migrasi memastikan tidak ada baris `payment_type = 4` (kartu) lebih dulu; jika ada,
    biarkan constraint tetap longgar / minta keputusan (kemungkinan besar belum ada data kartu
    karena fitur kartu baru saja ditambahkan).
  - Constraint `product_type IN (0,1)`.
  - Kolom VA pada `payments` (`va_bank`, `va_number`, `bill_key`, `biller_code`) bila belum ada.
  - `-- +goose Down` mengembalikan semua perubahan.

## Alur Data (contoh)

**POS jual pulsa + bayar VA BCA:**
1. `POST /transactions {type_payment:"transfer", bank:"bca", details:[{product_id:<pulsa>, qty:1, destination:"0812..."}]}`.
2. Usecase: produk pulsa = digital → skip stok, set destination, validasi bank=bca.
3. Repo simpan transaksi (stok fisik tidak berubah), `payment_type=2`, `bank=bca`.

**Online VA Mandiri (barang fisik):**
1. `POST /api/payments/va {bank:"mandiri", items:[...]}`.
2. `applyFee("va", subtotal)` +Rp4.000; reserve stok; `gateway.ChargeVA` → echannel →
   `bill_key`+`biller_code`.
3. Payment `method=va, va_bank=mandiri`; webhook menyelesaikan; `finalizeSuccess` buat transaksi
   `payment_type=transfer, bank=mandiri`.

## Error Handling

Pola tetap: charge gagal → `releaseStock` + error; signature webhook invalid → 403; gagal
finalisasi setelah settle → log `PAYMENT_RECONCILIATION_REQUIRED`. Validasi baru (bank wajib untuk
transfer, destination wajib untuk digital) ditolak 400 di layer usecase/DTO sebelum menyentuh DB/gateway.

## Testing

- Unit: `applyFee` (QRIS 0,7% bulat ke atas, VA flat).
- Unit: `mapChargeResponse` VA BCA (`va_numbers`) & Mandiri (echannel `bill_key`).
- Unit: enum `MoneyPayment` tanpa kartu, `ProductType` parse/string.
- Unit/usecase: `addTransaction` produk digital (destination wajib, stok tidak dipotong) & VA (bank wajib).

## Keputusan Terbuka / Asumsi

1. **Data kartu lama:** diasumsikan belum ada `payment_type = 4` di produksi (fitur kartu baru).
   Jika ternyata ada, constraint tidak diperketat & butuh keputusan terpisah.
2. **Sub-jenis digital** (DANA vs GoPay vs pulsa vs paket) memakai `Category` string, bukan enum
   khusus. Jika perlu pelaporan per-jenis, bisa ditambah enum `DigitalCategory` menyusul.
3. **Denominasi** (nominal saldo/pulsa) diwakili oleh `SellingPrice` produk (mis. produk
   "Pulsa 10.000") — tidak ada field nominal terpisah.
