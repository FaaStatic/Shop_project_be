# Payment Methods (Cash/QRIS/VA BCA+Mandiri) & Digital Products Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Simplify payment methods to Cash / QRIS / Virtual Account (BCA + Mandiri) while keeping debt (hutang), remove card & standalone-GoPay, and add digital catalog products (e-wallet top-up, pulsa, data packages) with a destination number and no stock deduction.

**Architecture:** Hexagonal layout already in place — `domain` holds models + ports, `usecase` holds business logic, `repository`/`infrastructure/api` hold adapters. Payment flows through the `domain.PaymentGateway` port (Midtrans Core API adapter). Enum numbering in `enum.MoneyPayment` is preserved (only `kartu` removed) so report SQL keyed on `payment_type = 1` and historical rows stay correct. Digital behavior is driven by a new `enum.ProductType` on `Products`; stock loops skip digital products.

**Tech Stack:** Go 1.26.1, GORM (PostgreSQL), goose migrations, Fiber v3, midtrans-go v1.3.8, zap, google/uuid, go-playground validator.

## Global Constraints

- Go module: `shop_project_be`, Go 1.26.1.
- Preserve `enum.MoneyPayment` numbering: `tunai=0, hutang=1, transfer=2, qris=3`. Do NOT renumber. `kartu` (was 4) is removed.
- Fees are added to gross for online charges only; percentages round UP (`math.Ceil`): QRIS 0.7%, VA (BCA & Mandiri) Rp4.000 flat.
- VA banks supported: `bca` (Midtrans `bank_transfer`) and `mandiri` (Midtrans `echannel`). No others.
- Online Midtrans channels: QRIS + VA only. No standalone GoPay charge.
- Digital products: `product_type` in {`physical`=0, `digital`=1}; digital products never reserve/deduct/restore stock; digital transaction lines require a `destination`.
- Follow existing doc-comment style (`// FuncName implements [domain.X].`, package-level rationale comments). Keep error strings lowercase, no trailing punctuation.
- Every task ends green (`go build ./...` + relevant `go test`), then a commit.

---

## File Structure

**Created:**
- `internal/constant/enum/type_product_type.go` — `ProductType` enum + `ParseProductType`/`String`.
- `internal/constant/enum/type_product_type_test.go` — enum unit tests.
- `internal/constant/enum/type_payment_test.go` — asserts `kartu` removed, numbering intact.
- `internal/usecase/payment_fee_test.go` — `applyFee` unit tests.
- `infrastructure/api/payment/midtrans_gateway_test.go` — `mapChargeResponse` VA mapping tests.
- `infrastructure/database/migrations/00005_payment_methods_and_digital_products.sql` — schema changes.

**Modified:**
- `internal/constant/enum/type_payment.go` — drop `kartu`.
- `internal/domain/product.go` — `Products.ProductType`; digital-aware doc.
- `internal/domain/transactions.go` — `Transactions.Bank`, `TransactionsDetail.Destination`, check constraint tag.
- `internal/domain/payment.go` — port: remove `ChargeCard`, add `ChargeVA`; input/result/model fields.
- `internal/dto/request_dto/product_request.go` — `ProductType` on Add/Update.
- `internal/dto/request_dto/transaction_request.go` — `Bank`, detail `Destination`, `oneof` updates.
- `internal/dto/request_dto/payment_request.go` — remove `ChargeCardRequest`, add `ChargeVARequest`.
- `internal/dto/response_dto/payment_response.go` — VA fields, method comment.
- `pkg/sheet/product_sheet.go` — parse optional `product_type` column.
- `internal/usecase/product_usecase.go` — bulk import sets `ProductType`.
- `internal/repository/product_repository.go` — digital-aware `ReserveStock`/`RestoreStock` (AddBulkProduct needs no query change).
- `internal/usecase/transaction_usecase.go` — destination + bank validation, digital pricing/stock.
- `internal/repository/transaction_repository.go` — skip digital in stock deduct/restore.
- `internal/usecase/payment_usecase.go` — remove `ChargeCard`, add `ChargeVA`, `applyFee`, `methodToPaymentType`, `finalizeSuccess` bank.
- `infrastructure/api/payment/midtrans_gateway.go` — remove `ChargeCard`, add `ChargeVA`, VA response mapping.
- `internal/delivery/http/handler/payment_handler.go` — remove `ChargeCard`, add `ChargeVA`.
- `internal/delivery/http/route/route.go` — remove `/payments/card`, add `/payments/va`.

---

## Task 1: Remove `kartu` from MoneyPayment enum

**Files:**
- Modify: `internal/constant/enum/type_payment.go`
- Modify: `internal/dto/request_dto/transaction_request.go`
- Test: `internal/constant/enum/type_payment_test.go` (create)

**Interfaces:**
- Produces: `enum.MoneyPayment` with values `tunai=0, hutang=1, transfer=2, qris=3`; `ParseMoneyPayment(s string) (MoneyPayment, error)` errors on `"kartu"`; `(MoneyPayment).String()` never returns `"kartu"`.

- [ ] **Step 1: Write the failing test**

Create `internal/constant/enum/type_payment_test.go`:

```go
package enum

import "testing"

func TestParseMoneyPayment_Valid(t *testing.T) {
	cases := map[string]MoneyPayment{
		"tunai": tunai, "hutang": hutang, "transfer": transfer, "qris": qris,
	}
	for in, want := range cases {
		got, err := ParseMoneyPayment(in)
		if err != nil || got != want {
			t.Fatalf("ParseMoneyPayment(%q) = %v, %v; want %v, nil", in, got, err, want)
		}
	}
}

func TestParseMoneyPayment_KartuRemoved(t *testing.T) {
	if _, err := ParseMoneyPayment("kartu"); err == nil {
		t.Fatal("expected error for removed 'kartu' payment type")
	}
}

func TestMoneyPaymentNumberingPreserved(t *testing.T) {
	if tunai != 0 || hutang != 1 || transfer != 2 || qris != 3 {
		t.Fatalf("numbering changed: tunai=%d hutang=%d transfer=%d qris=%d", tunai, hutang, transfer, qris)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/constant/enum/ -run TestParseMoneyPayment_KartuRemoved -v`
Expected: FAIL (currently `"kartu"` parses without error).

- [ ] **Step 3: Edit the enum to drop `kartu`**

In `internal/constant/enum/type_payment.go`, remove the `kartu` constant, its `String()` case, and its `ParseMoneyPayment` case. Result:

```go
const (
	tunai MoneyPayment = iota
	hutang
	transfer
	qris
)

func (typeItem MoneyPayment) String() string {
	switch typeItem {
	case tunai:
		return "tunai"
	case hutang:
		return "hutang"
	case transfer:
		return "transfer"
	case qris:
		return "qris"
	default:
		return "unknown"
	}
}

func ParseMoneyPayment(moneyPaymentStr string) (MoneyPayment, error) {
	switch strings.ToLower(moneyPaymentStr) {
	case "tunai":
		return tunai, nil
	case "hutang":
		return hutang, nil
	case "transfer":
		return transfer, nil
	case "qris":
		return qris, nil
	default:
		return 0, errors.New("type payment not valid")
	}
}
```

- [ ] **Step 4: Update transaction DTO validation**

In `internal/dto/request_dto/transaction_request.go`, change the `AddTransactionRequest.TypePayment` tag from `oneof=tunai hutang transfer qris kartu` to `oneof=tunai hutang transfer qris`. (The `FilterTransactionRequest.TypePayment` is already `oneof=0 1 2 3`.)

- [ ] **Step 5: Run tests + build**

Run: `go test ./internal/constant/enum/ -v && go build ./...`
Expected: PASS. Build may fail in `payment_usecase.go` (references `"kartu"`) — that is fixed in Task 9; if building the whole module fails only there, proceed (this task's package tests pass). To keep the tree green, temporarily leave `payment_usecase.go` untouched here — it still compiles because it only produces the string `"kartu"`, it does not reference the enum constant. Confirm `go build ./...` passes.

- [ ] **Step 6: Commit**

```bash
git add internal/constant/enum/type_payment.go internal/constant/enum/type_payment_test.go internal/dto/request_dto/transaction_request.go
git commit -m "refactor(enum): remove kartu payment type, keep numbering"
```

---

## Task 2: Add ProductType enum

**Files:**
- Create: `internal/constant/enum/type_product_type.go`
- Test: `internal/constant/enum/type_product_type_test.go`

**Interfaces:**
- Produces: `enum.ProductType` with `Physical ProductType = 0`, `Digital ProductType = 1` (exported, because usecase/domain in other packages must reference them); `ParseProductType(s string) (ProductType, error)` (accepts `"", "physical", "0"` → Physical; `"digital", "1"` → Digital); `(ProductType).String()` → `"physical"`/`"digital"`; `(ProductType).IsDigital() bool`.

- [ ] **Step 1: Write the failing test**

Create `internal/constant/enum/type_product_type_test.go`:

```go
package enum

import "testing"

func TestParseProductType(t *testing.T) {
	cases := map[string]ProductType{
		"": Physical, "physical": Physical, "0": Physical,
		"digital": Digital, "1": Digital, "DIGITAL": Digital,
	}
	for in, want := range cases {
		got, err := ParseProductType(in)
		if err != nil || got != want {
			t.Fatalf("ParseProductType(%q) = %v, %v; want %v, nil", in, got, err, want)
		}
	}
}

func TestParseProductType_Invalid(t *testing.T) {
	if _, err := ParseProductType("gas"); err == nil {
		t.Fatal("expected error for invalid product type")
	}
}

func TestProductTypeIsDigital(t *testing.T) {
	if !Digital.IsDigital() || Physical.IsDigital() {
		t.Fatal("IsDigital mismatch")
	}
	if Physical.String() != "physical" || Digital.String() != "digital" {
		t.Fatal("String mismatch")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/constant/enum/ -run TestParseProductType -v`
Expected: FAIL (undefined `ProductType`).

- [ ] **Step 3: Create the enum**

Create `internal/constant/enum/type_product_type.go`:

```go
package enum

import (
	"errors"
	"strings"
)

// ProductType distinguishes physical inventory from digital goods
// (e-wallet top-up, pulsa, data packages). Digital products are not
// stock-managed and require a destination (phone/account) at sale time.
type ProductType int

const (
	// Physical is stock-managed inventory (default).
	Physical ProductType = iota
	// Digital is a non-stock good fulfilled to a destination number/account.
	Digital
)

func (p ProductType) String() string {
	switch p {
	case Physical:
		return "physical"
	case Digital:
		return "digital"
	default:
		return "unknown"
	}
}

// IsDigital reports whether this product bypasses stock management.
func (p ProductType) IsDigital() bool { return p == Digital }

// ParseProductType accepts a number ("0"/"1") or text ("physical"/"digital").
// Empty defaults to Physical.
func ParseProductType(s string) (ProductType, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "physical", "0":
		return Physical, nil
	case "digital", "1":
		return Digital, nil
	default:
		return 0, errors.New("invalid product type")
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/constant/enum/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/constant/enum/type_product_type.go internal/constant/enum/type_product_type_test.go
git commit -m "feat(enum): add ProductType (physical/digital)"
```

---

## Task 3: Domain fields — Products.ProductType, Transactions.Bank, TransactionsDetail.Destination

**Files:**
- Modify: `internal/domain/product.go:15-30`
- Modify: `internal/domain/transactions.go:15-43`
- Modify: `internal/dto/request_dto/product_request.go`
- Modify: `internal/dto/request_dto/transaction_request.go`

**Interfaces:**
- Produces: `domain.Products.ProductType enum.ProductType`; `domain.Transactions.Bank *string`; `domain.TransactionsDetail.Destination *string`. DTOs: `AddProduct.ProductType *int`, `UpdateProduct.ProductType *int`, `AddTransactionRequest.Bank *string`, `AddTransactionDetailRequest.Destination *string`.

- [ ] **Step 1: Add ProductType to the Products model**

In `internal/domain/product.go`, add the field after `Unit` (keep GORM check constraint):

```go
	Unit             enum.ProductUnit `gorm:"type:smallint;check:unit IN (0,1,2,3,4,5);not null" json:"unit"`
	ProductType      enum.ProductType `gorm:"column:product_type;type:smallint;check:product_type IN (0,1);not null;default:0" json:"product_type"`
```

- [ ] **Step 2: Add Bank + Destination to transaction models**

In `internal/domain/transactions.go`, update the `PaymentType` check constraint and add `Bank`:

```go
	PaymentType      enum.MoneyPayment `gorm:"type:smallint;check:payment_type IN (0,1,2,3);not null" json:"payment_type"`
	Bank             *string           `gorm:"type:varchar(20)" json:"bank,omitempty"` // "bca"|"mandiri", set only when PaymentType == transfer
```

In `TransactionsDetail`, add after `Subtotal`:

```go
	Destination *string `gorm:"type:varchar(50)" json:"destination,omitempty"` // phone/e-wallet account for digital products
```

- [ ] **Step 3: Add DTO fields**

In `internal/dto/request_dto/product_request.go`, add to `AddProduct` and `UpdateProduct`:

```go
// AddProduct:
	ProductType int `json:"product_type,omitempty" validate:"omitempty,oneof=0 1"`
// UpdateProduct:
	ProductType *int `json:"product_type,omitempty" validate:"omitempty,oneof=0 1"`
```

In `internal/dto/request_dto/transaction_request.go`, add `Bank` to `AddTransactionRequest` and `Destination` to `AddTransactionDetailRequest`:

```go
// AddTransactionRequest (after CustomerId):
	Bank *string `json:"bank,omitempty" validate:"omitempty,oneof=bca mandiri"`
// AddTransactionDetailRequest (after Qty):
	Destination *string `json:"destination,omitempty"`
```

- [ ] **Step 4: Build**

Run: `go build ./...`
Expected: PASS (fields are additive; no callers broken yet).

- [ ] **Step 5: Commit**

```bash
git add internal/domain/product.go internal/domain/transactions.go internal/dto/request_dto/product_request.go internal/dto/request_dto/transaction_request.go
git commit -m "feat(domain): add ProductType, transaction Bank & detail Destination"
```

---

## Task 4: Bulk import — parse and persist product_type

**Files:**
- Modify: `pkg/sheet/product_sheet.go:17-27, 129-137`
- Modify: `internal/usecase/product_usecase.go:71-98`
- Test: `pkg/sheet/product_sheet_test.go` (add one case)

**Interfaces:**
- Consumes: `enum.ParseProductType` (Task 2), `domain.Products.ProductType` (Task 3).
- Produces: `sheet.ProductRow.ProductType string` (raw); bulk-built `domain.Products` carry parsed `ProductType`.

- [ ] **Step 1: Write the failing test**

In `pkg/sheet/product_sheet_test.go`, add:

```go
func TestParseProducts_ProductTypeColumn(t *testing.T) {
	csvData := "sku,product_name,unit,purchase_price,selling_price,selling_price_debt,stock,category,product_type\n" +
		"SKU-D,Pulsa 10k,pcs,10000,11000,11000,0,pulsa,digital\n" +
		"SKU-P,Beras,kg,60000,65000,67000,10,sembako,\n"
	rows, _, err := ParseProducts(strings.NewReader(csvData), "products.csv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("want 2 rows, got %d", len(rows))
	}
	if rows[0].ProductType != "digital" {
		t.Fatalf("want digital, got %q", rows[0].ProductType)
	}
	if rows[1].ProductType != "" {
		t.Fatalf("want empty (defaults later), got %q", rows[1].ProductType)
	}
}
```

Ensure `strings` is imported in the test file (add if missing).

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/sheet/ -run TestParseProducts_ProductTypeColumn -v`
Expected: FAIL (`rows[0].ProductType` undefined).

- [ ] **Step 3: Add the field + parse it**

In `pkg/sheet/product_sheet.go`, add to `ProductRow` (after `Image`):

```go
	ProductType string // raw; parsed by the caller. Empty -> physical.
```

In the row builder (where `get(...)` is used), add:

```go
		row := ProductRow{
			Line:        line,
			SKU:         get("sku"),
			ProductName: get("product_name"),
			Unit:        get("unit"),
			Category:    get("category"),
			Image:       get("image"),
			ProductType: get("product_type"),
		}
```

(`product_type` is NOT added to `requiredHeaders`, so old files without the column still parse.)

- [ ] **Step 4: Set ProductType during bulk build**

In `internal/usecase/product_usecase.go`, inside `AddBulkProductShopWithLock`, in the loop after the unit parse, add a product-type parse and set the field:

```go
		unit, err := enum.ParseProductUnit(row.Unit)
		if err != nil {
			rowErrors = append(rowErrors, sheet.RowError{Line: row.Line, Message: "invalid unit"})
			continue
		}
		productType, err := enum.ParseProductType(row.ProductType)
		if err != nil {
			rowErrors = append(rowErrors, sheet.RowError{Line: row.Line, Message: "invalid product_type"})
			continue
		}
		products = append(products, &domain.Products{
			SKU:              row.SKU,
			ProductName:      row.ProductName,
			Unit:             unit,
			ProductType:      productType,
			PurchasePrice:    row.PurchasePrice,
			SellingPrice:     row.SellingPrice,
			SellingPriceDebt: row.SellingPriceDebt,
			Stock:            row.Stock,
			Category:         row.Category,
			Image:            row.Image,
		})
```

- [ ] **Step 5: Set ProductType for single Add too**

In the same file, `AddProductShopWithLock`, add `ProductType: enum.ProductType(request.ProductType),` to the `domain.Products{...}` literal (right after `Unit:`).

- [ ] **Step 6: Run tests + build**

Run: `go test ./pkg/sheet/ ./internal/usecase/... && go build ./...`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add pkg/sheet/product_sheet.go pkg/sheet/product_sheet_test.go internal/usecase/product_usecase.go
git commit -m "feat(product): bulk & single import set product_type"
```

---

## Task 5: Digital-aware stock in product repository

**Files:**
- Modify: `internal/repository/product_repository.go:249-303`

**Interfaces:**
- Consumes: `domain.Products.ProductType`, `enum.Digital`.
- Produces: `ReserveStock`/`RestoreStock` skip digital products (no lock-and-deduct for them).

Note: `AddBulkProduct` needs NO query change — GORM writes the `product_type` column from the struct; `physical=0` matches the column default. Do not modify it.

- [ ] **Step 1: Skip digital in ReserveStock**

In `internal/repository/product_repository.go`, `ReserveStock`, inside the per-item loop after the product row is locked, add a skip before the stock check:

```go
			if product.ProductType.IsDigital() {
				// Digital goods (pulsa / e-wallet / data) are not stock-managed.
				continue
			}
			if product.Stock < it.Qty {
```

- [ ] **Step 2: Skip digital in RestoreStock**

`RestoreStock` currently does a blind `stock + qty` update without loading the row. Change it to load the product first so digital items are skipped:

```go
		for _, it := range items {
			var product domain.Products
			if err := tx.Where("id = ?", it.ProductID).First(&product).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					continue // row gone; nothing to restore (matches prior lock-free behavior)
				}
				return fmt.Errorf("failed to load product for restore: %w", err)
			}
			if product.ProductType.IsDigital() {
				continue
			}
			if err := tx.Model(&domain.Products{}).Where("id = ?", it.ProductID).
				Update("stock", gorm.Expr("stock + ?", it.Qty)).Error; err != nil {
				return fmt.Errorf("failed to restore stock: %w", err)
			}
		}
```

(The `lockProductsOrdered` call above the loop stays; it already locked these rows FOR UPDATE.)

- [ ] **Step 2b: Add a fake-DB-free reasoning check**

No unit DB test exists for this repo (it needs Postgres). Verify by build + the existing test suite instead. Manual reasoning note to include in the commit body: digital products have `ProductType==Digital`, so both paths `continue` before any stock write.

- [ ] **Step 3: Build + existing tests**

Run: `go build ./... && go test ./internal/repository/... 2>&1 | tail -20`
Expected: build PASS; repository tests pass or are skipped (no new failures).

- [ ] **Step 4: Commit**

```bash
git add internal/repository/product_repository.go
git commit -m "feat(product): ReserveStock/RestoreStock skip digital products"
```

---

## Task 6: Transaction repository — skip digital in deduct/restore

**Files:**
- Modify: `internal/repository/transaction_repository.go:43-118, 123-204`

**Interfaces:**
- Consumes: `domain.Products.ProductType`, `enum.Digital`.
- Produces: `CreateTransaction` does not deduct stock for digital lines; `DeleteTransaction` does not restore stock for digital lines.

- [ ] **Step 1: Skip digital in CreateTransaction deduct loop**

In `CreateTransaction`, inside the `if deductStock {` block's per-detail loop, after the product is locked-and-loaded, add:

```go
				if product.ProductType.IsDigital() {
					continue // digital goods are not stock-managed
				}
				qty := d.Qty
				if product.Stock < qty {
```

- [ ] **Step 2: Skip digital in DeleteTransaction restore loop**

In `DeleteTransaction`, inside the restore loop after the product is locked-and-loaded, add before the `Update("stock", ...)`:

```go
				if product.ProductType.IsDigital() {
					continue
				}
				if err := tx.Model(&domain.Products{}).Where("id = ?", d.ProductID).
					Update("stock", product.Stock+d.Qty).Error; err != nil {
```

- [ ] **Step 3: Build**

Run: `go build ./...`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/repository/transaction_repository.go
git commit -m "feat(transaction): skip stock deduct/restore for digital products"
```

---

## Task 7: Transaction usecase — bank & destination validation, digital pricing

**Files:**
- Modify: `internal/usecase/transaction_usecase.go:72-191`

**Interfaces:**
- Consumes: `enum.Digital`, `AddTransactionRequest.Bank`, `AddTransactionDetailRequest.Destination`, `Transactions.Bank`, `TransactionsDetail.Destination`.
- Produces: transaction persisted with `Bank` (for transfer) and per-line `Destination` (for digital); digital lines priced at `SellingPrice`, skip the pre-check `product.Stock < qty`.

- [ ] **Step 1: Validate bank vs payment type**

In `addTransaction`, after `paymentType, err := enum.ParseMoneyPayment(...)` and `isHutang := ...`, add:

```go
	isTransfer := paymentType.String() == "transfer"
	var bank *string
	if isTransfer {
		if dto.Bank == nil || (*dto.Bank != "bca" && *dto.Bank != "mandiri") {
			t.log.Error("bank is required for transfer", zap.String("no_invoice", noInvoice))
			return fmt.Errorf("bank (bca/mandiri) is required for transfer payment")
		}
		bank = dto.Bank
	}
```

- [ ] **Step 2: Digital pricing + destination in the detail loop**

Replace the body of the `for _, detail := range dto.Details` loop's pricing section so digital products are priced at selling price, require a destination, and carry it:

```go
		unitPrice := product.SellingPrice
		if isHutang && !product.ProductType.IsDigital() {
			unitPrice = product.SellingPriceDebt
		}

		var destination *string
		if product.ProductType.IsDigital() {
			if detail.Destination == nil || strings.TrimSpace(*detail.Destination) == "" {
				t.log.Error("destination required for digital product", zap.String("product_id", detail.ProductId))
				return fmt.Errorf("destination is required for digital product %s", detail.ProductId)
			}
			d := strings.TrimSpace(*detail.Destination)
			destination = &d
		}

		subtotal := unitPrice * detail.Qty
		total += subtotal

		detailTrx = append(detailTrx, domain.TransactionsDetail{
			ProductID:   productId,
			Price:       unitPrice,
			PriceDebt:   product.SellingPriceDebt,
			Qty:         detail.Qty,
			Subtotal:    subtotal,
			Destination: destination,
		})
```

- [ ] **Step 3: Set Bank on the persisted transaction**

In the `data := &domain.Transactions{...}` literal, add `Bank: bank,` after `PaymentType: paymentType,`.

- [ ] **Step 4: Build**

Run: `go build ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/usecase/transaction_usecase.go
git commit -m "feat(transaction): validate bank for transfer, destination for digital"
```

---

## Task 8: Payment domain port — remove ChargeCard, add ChargeVA

**Files:**
- Modify: `internal/domain/payment.go:39-125`

**Interfaces:**
- Produces:
  - `Payment` fields: `Method` doc `"qris" | "va"`; add `VABank string`, `VANumber string`, `BillKey string`, `BillerCode string`.
  - `GatewayChargeInput`: remove `CardTokenID`, `Authentication`; add `Bank string`.
  - `GatewayChargeResult`: add `VANumber string`, `Bank string`, `BillKey string`, `BillerCode string`.
  - `PaymentGateway` interface: remove `ChargeCard`; add `ChargeVA(ctx, in) (*GatewayChargeResult, error)`.

- [ ] **Step 1: Update the Payment model**

In `internal/domain/payment.go`, change the `Method` comment and add VA fields after `RedirectURL`:

```go
	Method      string        `gorm:"type:varchar(20);not null" json:"method"` // "qris" | "va"
```

```go
	// VA (Virtual Account) details, filled for method == "va".
	VABank     string `gorm:"column:va_bank;type:varchar(20)" json:"va_bank,omitempty"`     // "bca"|"mandiri"
	VANumber   string `gorm:"column:va_number;type:varchar(50)" json:"va_number,omitempty"` // BCA bank_transfer VA
	BillKey    string `gorm:"column:bill_key;type:varchar(50)" json:"bill_key,omitempty"`   // Mandiri echannel
	BillerCode string `gorm:"column:biller_code;type:varchar(20)" json:"biller_code,omitempty"`
```

- [ ] **Step 2: Update gateway input/result + interface**

Replace the card-only fields in `GatewayChargeInput`:

```go
type GatewayChargeInput struct {
	OrderID     string
	GrossAmount int64
	Items       []GatewayItem
	Customer    GatewayCustomer

	// VA-only: "bca" (bank_transfer) or "mandiri" (echannel).
	Bank string
}
```

Add VA fields to `GatewayChargeResult` (after `RedirectURL`):

```go
	VANumber   string
	Bank       string
	BillKey    string
	BillerCode string
```

Update the interface:

```go
type PaymentGateway interface {
	ChargeQris(ctx context.Context, in GatewayChargeInput) (*GatewayChargeResult, error)
	ChargeVA(ctx context.Context, in GatewayChargeInput) (*GatewayChargeResult, error)
	CheckStatus(ctx context.Context, orderID string) (*GatewayChargeResult, error)
	VerifySignature(orderID, statusCode, grossAmount, signatureKey string) bool
}
```

- [ ] **Step 3: Build (expected to fail in adapter/usecase)**

Run: `go build ./... 2>&1 | head`
Expected: FAIL in `midtrans_gateway.go` (still has `ChargeCard`, no `ChargeVA`) and `payment_usecase.go`. Fixed in Tasks 8-adapter/9. This step just confirms the port compiles in isolation:

Run: `go vet ./internal/domain/ 2>&1 | head`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/domain/payment.go
git commit -m "feat(payment): port supports VA (bca/mandiri), drop card"
```

---

## Task 9: Midtrans adapter — ChargeVA + VA response mapping

**Files:**
- Modify: `infrastructure/api/payment/midtrans_gateway.go`
- Test: `infrastructure/api/payment/midtrans_gateway_test.go` (create)

**Interfaces:**
- Consumes: `domain.GatewayChargeInput.Bank`, result VA fields (Task 8).
- Produces: `(*midtransGateway).ChargeVA`; `mapChargeResponse` fills `VANumber`/`Bank` (from `va_numbers[0]`) and `BillKey`/`BillerCode` (echannel).

- [ ] **Step 1: Write the failing test**

Create `infrastructure/api/payment/midtrans_gateway_test.go`:

```go
package payment

import (
	"testing"

	"github.com/midtrans/midtrans-go/coreapi"
)

func TestMapChargeResponse_BCAVirtualAccount(t *testing.T) {
	res := &coreapi.ChargeResponse{
		TransactionID:     "txn-1",
		OrderID:           "INV-1",
		PaymentType:       "bank_transfer",
		TransactionStatus: "pending",
		StatusCode:        "201",
		VaNumbers:         []coreapi.VANumber{{Bank: "bca", VANumber: "12345678"}},
	}
	out := mapChargeResponse(res)
	if out.VANumber != "12345678" || out.Bank != "bca" {
		t.Fatalf("VA mapping failed: %+v", out)
	}
}

func TestMapChargeResponse_MandiriEChannel(t *testing.T) {
	res := &coreapi.ChargeResponse{
		TransactionID:     "txn-2",
		OrderID:           "INV-2",
		PaymentType:       "echannel",
		TransactionStatus: "pending",
		StatusCode:        "201",
		BillKey:           "BK-9",
		BillerCode:        "BC-7",
	}
	out := mapChargeResponse(res)
	if out.BillKey != "BK-9" || out.BillerCode != "BC-7" {
		t.Fatalf("echannel mapping failed: %+v", out)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./infrastructure/api/payment/ -run TestMapChargeResponse -v`
Expected: FAIL (fields not populated / build error until Task 8 merged — Task 8 is merged, so it compiles and asserts fail).

- [ ] **Step 3: Extend mapChargeResponse**

In `mapChargeResponse`, after the existing QR-URL loop, add VA extraction:

```go
	// Bank transfer (BCA/BNI/BRI...) exposes a va_numbers array.
	if len(res.VaNumbers) > 0 {
		out.VANumber = res.VaNumbers[0].VANumber
		out.Bank = res.VaNumbers[0].Bank
	}
	// Mandiri echannel uses bill_key + biller_code instead of a VA number.
	out.BillKey = res.BillKey
	out.BillerCode = res.BillerCode
	return out
```

(Replace the current trailing `return out` with the block above.)

- [ ] **Step 4: Replace ChargeCard with ChargeVA**

Delete the entire `ChargeCard` method. Add:

```go
// ChargeVA creates a Virtual Account charge. BCA uses the bank_transfer flow
// (a va_number is returned); Mandiri uses echannel (bill_key + biller_code).
func (g *midtransGateway) ChargeVA(_ context.Context, in domain.GatewayChargeInput) (*domain.GatewayChargeResult, error) {
	req := &coreapi.ChargeReq{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  in.OrderID,
			GrossAmt: in.GrossAmount,
		},
		CustomerDetails: toCustomerDetails(in.Customer),
	}
	switch strings.ToLower(in.Bank) {
	case "bca":
		req.PaymentType = coreapi.PaymentTypeBankTransfer
		req.BankTransfer = &coreapi.BankTransferDetails{Bank: midtrans.BankBca}
	case "mandiri":
		req.PaymentType = coreapi.PaymentTypeEChannel
		req.EChannel = &coreapi.EChannelDetail{
			BillInfo1: "Payment",
			BillInfo2: in.OrderID,
		}
	default:
		return nil, errors.New("unsupported va bank")
	}
	if items := toItemDetails(in.Items); len(items) > 0 {
		req.Items = &items
	}
	res, mErr := g.client.ChargeTransaction(req)
	if mErr != nil {
		return nil, errors.New(mErr.GetMessage())
	}
	out := mapChargeResponse(res)
	if out.Bank == "" {
		out.Bank = strings.ToLower(in.Bank) // echannel has no va_numbers[].Bank
	}
	return out, nil
}
```

- [ ] **Step 5: Run test + build**

Run: `go test ./infrastructure/api/payment/ -v && go build ./... 2>&1 | head`
Expected: adapter tests PASS; module build still fails only in `payment_usecase.go` (Task 10 fixes it).

- [ ] **Step 6: Commit**

```bash
git add infrastructure/api/payment/midtrans_gateway.go infrastructure/api/payment/midtrans_gateway_test.go
git commit -m "feat(midtrans): ChargeVA (bca bank_transfer, mandiri echannel) + VA mapping"
```

---

## Task 10: Payment usecase — ChargeVA, applyFee, method mapping

**Files:**
- Modify: `internal/usecase/payment_usecase.go`
- Test: `internal/usecase/payment_fee_test.go` (create)

**Interfaces:**
- Consumes: `gateway.ChargeVA`, `GatewayChargeInput.Bank`, `Payment.VABank/VANumber/BillKey/BillerCode`, `ChargeVARequest` (Task 11).
- Produces: `(*paymentUsecase).ChargeVA`; `applyFee(method string, subtotal int64) int64`; `methodToPaymentType("va") == "transfer"`; `newPayment` sets VA fields.

- [ ] **Step 1: Write the failing fee test**

Create `internal/usecase/payment_fee_test.go`:

```go
package usecase

import "testing"

func TestApplyFee(t *testing.T) {
	cases := []struct {
		method   string
		subtotal int64
		want     int64
	}{
		{"qris", 100000, 100700},  // +0.7%
		{"qris", 10001, 10072},    // 70.007 -> ceil 71 -> 10072
		{"va", 100000, 104000},    // +Rp4.000 flat
		{"cash", 100000, 100000},  // unknown method: no fee
	}
	for _, c := range cases {
		if got := applyFee(c.method, c.subtotal); got != c.want {
			t.Fatalf("applyFee(%q,%d)=%d want %d", c.method, c.subtotal, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/usecase/ -run TestApplyFee -v`
Expected: FAIL (undefined `applyFee`).

- [ ] **Step 3: Add applyFee + fee constants**

In `internal/usecase/payment_usecase.go` (near the other helpers), add:

```go
// Midtrans standard per-transaction fees, passed on to the buyer. Percentages
// are rounded UP to the nearest rupiah so the merchant never absorbs a fraction.
// Source: Midtrans pricing page (QRIS 0.7%, Virtual Account Rp4.000 flat).
const (
	feeQrisRate   = 0.007
	feeVAFlat     = 4000
)

// applyFee returns gross = subtotal + channel fee for online charges.
func applyFee(method string, subtotal int64) int64 {
	switch method {
	case "qris":
		return subtotal + int64(math.Ceil(float64(subtotal)*feeQrisRate))
	case "va":
		return subtotal + feeVAFlat
	default:
		return subtotal
	}
}
```

- [ ] **Step 4: Run fee test**

Run: `go test ./internal/usecase/ -run TestApplyFee -v`
Expected: PASS.

- [ ] **Step 5: Apply fee in ChargeQris + add ChargeVA**

In `ChargeQris`, after `gross, items, err := u.buildOrder(...)`, wrap gross with the fee:

```go
	gross = applyFee("qris", gross)
```

Replace the whole `ChargeCard` method with `ChargeVA`:

```go
// ChargeVA creates a Virtual Account payment (BCA bank_transfer or Mandiri
// echannel). The buyer pays the displayed VA number / bill; the final status
// arrives via webhook (see HandleNotification).
func (u *paymentUsecase) ChargeVA(ctx context.Context, request *requestdto.ChargeVARequest) (*responsedto.ChargePaymentResponse, error) {
	userID, err := uuid.Parse(request.UserId)
	if err != nil {
		u.log.Error("failed to parse user id", zap.Error(err))
		return nil, fmt.Errorf("invalid user id format")
	}
	customerID, err := parseOptionalUUID(request.CustomerId)
	if err != nil {
		u.log.Error("failed to parse customer id", zap.Error(err))
		return nil, fmt.Errorf("invalid customer id format")
	}

	gross, items, err := u.buildOrder(ctx, toItemPairs(request.Items))
	if err != nil {
		return nil, err
	}
	gross = applyFee("va", gross)

	orderID, err := u.resolveOrderID(ctx, request.NoInvoice)
	if err != nil {
		return nil, err
	}

	// Reserve stock BEFORE charging (see ChargeQris).
	if err := u.productRepo.ReserveStock(ctx, items); err != nil {
		u.log.Warn("failed to reserve stock", zap.Error(err), zap.String("order_id", orderID))
		return nil, fmt.Errorf("insufficient stock")
	}

	result, err := u.gateway.ChargeVA(ctx, domain.GatewayChargeInput{
		OrderID:     orderID,
		GrossAmount: gross,
		Bank:        request.Bank,
	})
	if err != nil {
		u.releaseStock(ctx, items, orderID)
		u.log.Error("failed to charge va", zap.Error(err))
		return nil, fmt.Errorf("failed to create va payment")
	}

	payment := u.newPayment(orderID, "va", userID, customerID, gross, items, result)
	payment.StockReserved = true
	return u.persistChargeResult(ctx, payment, result)
}
```

- [ ] **Step 6: Populate VA fields in newPayment + fix method mapping**

In `newPayment`, add VA fields to the returned `&domain.Payment{...}`:

```go
		VABank:     result.Bank,
		VANumber:   result.VANumber,
		BillKey:    result.BillKey,
		BillerCode: result.BillerCode,
```

Replace `methodToPaymentType`:

```go
// methodToPaymentType maps the online payment method to the internal POS
// payment enum string. VA settles as a bank transfer; QRIS stays qris.
func methodToPaymentType(method string) string {
	if method == "va" {
		return "transfer"
	}
	return "qris"
}
```

- [ ] **Step 7: Carry bank into the created transaction**

In `finalizeSuccess`, where `addReq := &requestdto.AddTransactionRequest{...}` is built, add the bank for VA:

```go
			var bank *string
			if payment.Method == "va" && payment.VABank != "" {
				b := payment.VABank
				bank = &b
			}
			addReq := &requestdto.AddTransactionRequest{
				NoInvoice:   payment.OrderID,
				TypePayment: methodToPaymentType(payment.Method),
				Bank:        bank,
				UserId:      payment.UserID.String(),
				CustomerId:  customerID,
				Details:     details,
			}
```

- [ ] **Step 8: Update the VA/QR response fields**

In `persistChargeResult`, the returned `ChargePaymentResponse` should include VA fields (added in Task 11). Add after `RedirectUrl`:

```go
		VaNumber:   payment.VANumber,
		Bank:       payment.VABank,
		BillKey:    payment.BillKey,
		BillerCode: payment.BillerCode,
```

- [ ] **Step 9: Build + tests**

Run: `go test ./internal/usecase/ && go build ./... 2>&1 | head`
Expected: usecase tests PASS; build fails only where handler/route/DTO still reference card (Task 11).

- [ ] **Step 10: Commit**

```bash
git add internal/usecase/payment_usecase.go internal/usecase/payment_fee_test.go
git commit -m "feat(payment): ChargeVA + applyFee (qris 0.7%, va flat), map va->transfer"
```

---

## Task 11: DTOs, handler, route — swap card for VA

**Files:**
- Modify: `internal/dto/request_dto/payment_request.go`
- Modify: `internal/dto/response_dto/payment_response.go`
- Modify: `internal/delivery/http/handler/payment_handler.go`
- Modify: `internal/delivery/http/route/route.go`

**Interfaces:**
- Consumes: `usecase.ChargeVA` (Task 10).
- Produces: `ChargeVARequest`; `ChargePaymentResponse` VA fields; `(*PaymentHandler).ChargeVA`; route `POST /api/payments/va`.

- [ ] **Step 1: Replace ChargeCardRequest with ChargeVARequest**

In `internal/dto/request_dto/payment_request.go`, delete `ChargeCardRequest` and add:

```go
// ChargeVARequest is used by Flutter for Virtual Account payments. Bank selects
// the VA channel: "bca" (bank_transfer) or "mandiri" (echannel).
type ChargeVARequest struct {
	UserId     string               `json:"user_id" validate:"required,uuid"`
	CustomerId *string              `json:"customer_id,omitempty" validate:"omitempty,uuid"`
	NoInvoice  string               `json:"no_invoice,omitempty"`
	Bank       string               `json:"bank" validate:"required,oneof=bca mandiri"`
	Items      []PaymentItemRequest `json:"items" validate:"required,min=1,dive"`
}
```

- [ ] **Step 2: Add VA fields + fix method comment in response DTO**

In `internal/dto/response_dto/payment_response.go`, change the `Method` comment to `"qris" | "va"` and add fields to `ChargePaymentResponse` (after `RedirectUrl`):

```go
	VaNumber   string `json:"va_number,omitempty"`   // BCA VA number
	Bank       string `json:"bank,omitempty"`        // "bca"|"mandiri"
	BillKey    string `json:"bill_key,omitempty"`    // Mandiri echannel
	BillerCode string `json:"biller_code,omitempty"` // Mandiri echannel
```

- [ ] **Step 3: Replace the ChargeCard handler with ChargeVA**

In `internal/delivery/http/handler/payment_handler.go`, delete the `ChargeCard` method and add:

```go
// ChargeVA godoc
//
//	@Summary		Create Virtual Account payment
//	@Description	Creates a VA charge via Midtrans. bank = "bca" (bank_transfer) or "mandiri" (echannel). Response contains va_number or bill_key/biller_code.
//	@Tags			Payments
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		requestdto.ChargeVARequest	true	"VA payment cart"
//	@Success		201		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/api/payments/va [post]
func (h *PaymentHandler) ChargeVA(c fiber.Ctx) error {
	var req requestdto.ChargeVARequest
	if err := bindBody(c, &req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request body", err)
	}
	req.UserId = middleware.GetUserID(c)
	if err := validate.Validate(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation failed", err)
	}
	res, err := h.usecase.ChargeVA(c.Context(), &req)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), err)
	}
	return response.Success(c, fiber.StatusCreated, "va payment created", res)
}
```

- [ ] **Step 4: Update the domain PaymentUsecase interface**

In `internal/domain/payment.go`, in the `PaymentUsecase` interface, replace the `ChargeCard` line:

```go
	ChargeVA(ctx context.Context, request *requestdto.ChargeVARequest) (*responsedto.ChargePaymentResponse, error)
```

- [ ] **Step 5: Swap the route**

In `internal/delivery/http/route/route.go`, replace line 65:

```go
		payments.Post("/va", h.Payment.ChargeVA)
```

- [ ] **Step 6: Build + full test**

Run: `go build ./... && go test ./...`
Expected: PASS across the module.

- [ ] **Step 7: Commit**

```bash
git add internal/dto/request_dto/payment_request.go internal/dto/response_dto/payment_response.go internal/delivery/http/handler/payment_handler.go internal/delivery/http/route/route.go internal/domain/payment.go
git commit -m "feat(payment): VA request/response/handler/route, drop card endpoint"
```

---

## Task 12: goose migration + final verification

**Files:**
- Create: `infrastructure/database/migrations/00005_payment_methods_and_digital_products.sql`

**Interfaces:**
- Consumes: all prior schema field expectations.
- Produces: DB columns `transactions.bank`, `transactions_detail.destination`, `products.product_type`, `payments.{va_bank,va_number,bill_key,biller_code}`; tightened check constraints.

- [ ] **Step 1: Write the migration**

Create `infrastructure/database/migrations/00005_payment_methods_and_digital_products.sql`:

```sql
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
```

- [ ] **Step 2: Verify migration SQL parses (dry check)**

Run: `grep -c "goose" infrastructure/database/migrations/00005_payment_methods_and_digital_products.sql`
Expected: `2` (Up + Down markers present). If a local Postgres + goose is available, run the project's migrate command; otherwise the SQL is applied on next app boot via `migrate_goose.go`.

- [ ] **Step 3: Full build, vet, test**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: all PASS.

- [ ] **Step 4: Regenerate swagger (if the project uses it)**

Run: `command -v swag >/dev/null && swag init -g cmd/*.go 2>/dev/null; echo done`
Expected: `done` (no-op if `swag` absent). Do not fail the task on swagger.

- [ ] **Step 5: Commit**

```bash
git add infrastructure/database/migrations/00005_payment_methods_and_digital_products.sql
git commit -m "feat(db): migration for payment methods & digital products"
```

- [ ] **Step 6: Final grep guard — no lingering card references**

Run: `grep -rn "ChargeCard\|ChargeCardRequest\|CardTokenID\|\"kartu\"\|/payments/card" --include="*.go" internal/ infrastructure/ | grep -v _test.go`
Expected: no output. If any remain, fix and amend the relevant commit.

---

## Self-Review Notes

- **Spec coverage:** Enum drop `kartu` (T1) ✓; ProductType (T2) ✓; domain fields (T3) ✓; bulk import (T4) ✓; digital stock product repo (T5) ✓; digital stock trx repo (T6) ✓; bank+destination validation (T7) ✓; payment port (T8) ✓; adapter VA (T9) ✓; usecase VA+fee (T10) ✓; DTO/handler/route (T11) ✓; migration (T12) ✓.
- **Fees:** QRIS 0.7% ceil, VA Rp4.000 flat — implemented in `applyFee` (T10), tested.
- **Numbering preserved:** T1 test asserts `tunai=0..qris=3`; report SQL untouched.
- **No standalone GoPay:** not present anywhere in the plan. ✓
- **VA banks BCA+Mandiri only:** adapter switch + DTO `oneof=bca mandiri` + usecase validation. ✓
```
