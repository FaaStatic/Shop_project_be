package sheet

import (
	"strings"
	"testing"
)

func TestParseProductsCSV(t *testing.T) {
	csvData := `sku,product_name,unit,purchase_price,selling_price,selling_price_debt,stock,category
SKU-1,Beras 5kg,kg,60000,65000,67000,10,sembako
SKU-2,Minyak 1L,2,15000,18000,19000,20,sembako
,No SKU,pcs,1000,2000,2100,5,lain
SKU-4,Bad Price,pcs,abc,2000,2100,5,lain

SKU-5,Gula 1kg,,10000,12000,12500,,sembako`

	rows, rowErrors, err := ParseProducts(strings.NewReader(csvData), "products.csv")
	if err != nil {
		t.Fatalf("unexpected fatal error: %v", err)
	}

	// Valid: SKU-1, SKU-2, SKU-5 -> 3 rows.
	if len(rows) != 3 {
		t.Fatalf("expected 3 valid rows, got %d (%+v)", len(rows), rows)
	}
	// Error: row without SKU + row with invalid price -> 2 errors.
	if len(rowErrors) != 2 {
		t.Fatalf("expected 2 row errors, got %d (%+v)", len(rowErrors), rowErrors)
	}

	// Check the column mapping is correct.
	if rows[0].SKU != "SKU-1" || rows[0].ProductName != "Beras 5kg" || rows[0].SellingPrice != 65000 {
		t.Fatalf("unexpected first row: %+v", rows[0])
	}
	// Empty stock -> 0.
	if rows[2].SKU != "SKU-5" || rows[2].Stock != 0 {
		t.Fatalf("unexpected last row: %+v", rows[2])
	}
}

func TestParseProductsMissingHeader(t *testing.T) {
	csvData := "sku,product_name,purchase_price\nSKU-1,Beras,60000"
	_, _, err := ParseProducts(strings.NewReader(csvData), "p.csv")
	if err == nil {
		t.Fatal("expected error for missing required header, got nil")
	}
}

func TestParseProductsUnsupportedFormat(t *testing.T) {
	_, _, err := ParseProducts(strings.NewReader("x"), "p.txt")
	if err == nil {
		t.Fatal("expected error for unsupported format, got nil")
	}
}
