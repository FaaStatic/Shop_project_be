// Package sheet membaca data produk dari file CSV atau Excel (.xlsx).
// Parser ini generic: tidak bergantung pada layer domain/usecase. Pemanggil
// yang memetakan hasilnya ke entitas aplikasi.
package sheet

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// ProductRow adalah satu baris produk hasil parsing (nilai mentah).
type ProductRow struct {
	Line             int // nomor baris pada file (1-based, termasuk header)
	SKU              string
	ProductName      string
	Unit             string // mentah; di-parse oleh pemanggil (angka atau teks)
	PurchasePrice    float64
	SellingPrice     float64
	SellingPriceDebt float64
	Stock            int
	Category         string
	Image            string
}

// RowError menandai baris yang gagal di-parse beserta alasannya.
type RowError struct {
	Line    int
	Message string
}

func (e RowError) Error() string {
	return fmt.Sprintf("baris %d: %s", e.Line, e.Message)
}

// kolom wajib pada file.
var requiredHeaders = []string{"sku", "product_name", "purchase_price", "selling_price", "selling_price_debt"}

// ParseProducts membaca produk dari r. Format ditentukan dari ekstensi filename
// (.csv atau .xlsx). Mengembalikan baris valid, daftar error per-baris (baris
// yang dilewati), dan error fatal (file tidak terbaca / header tidak lengkap).
func ParseProducts(r io.Reader, filename string) ([]ProductRow, []RowError, error) {
	ext := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(ext, ".csv"):
		return parseRecords(readCSV(r))
	case strings.HasSuffix(ext, ".xlsx"):
		records, err := readXLSX(r)
		if err != nil {
			return nil, nil, err
		}
		return parseRecords(records, nil)
	default:
		return nil, nil, fmt.Errorf("format file tidak didukung (gunakan .csv atau .xlsx)")
	}
}

func readCSV(r io.Reader) ([][]string, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1 // izinkan jumlah kolom bervariasi
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("gagal membaca csv: %w", err)
	}
	return records, nil
}

func readXLSX(r io.Reader) ([][]string, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca excel: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("file excel tidak memiliki sheet")
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("gagal membaca baris excel: %w", err)
	}
	return rows, nil
}

// parseRecords memetakan baris mentah (records) menjadi []ProductRow
// berdasarkan header pada baris pertama. Parameter err meneruskan error baca.
func parseRecords(records [][]string, err error) ([]ProductRow, []RowError, error) {
	if err != nil {
		return nil, nil, err
	}
	if len(records) == 0 {
		return nil, nil, fmt.Errorf("file kosong")
	}

	// Petakan nama header -> index kolom.
	colIndex := make(map[string]int)
	for i, h := range records[0] {
		colIndex[normalizeHeader(h)] = i
	}
	for _, h := range requiredHeaders {
		if _, ok := colIndex[h]; !ok {
			return nil, nil, fmt.Errorf("kolom wajib '%s' tidak ditemukan di header", h)
		}
	}

	var rows []ProductRow
	var rowErrors []RowError

	for i := 1; i < len(records); i++ {
		line := i + 1 // 1-based, header = baris 1
		rec := records[i]
		if isEmptyRecord(rec) {
			continue
		}

		get := func(key string) string {
			idx, ok := colIndex[key]
			if !ok || idx >= len(rec) {
				return ""
			}
			return strings.TrimSpace(rec[idx])
		}

		row := ProductRow{
			Line:        line,
			SKU:         get("sku"),
			ProductName: get("product_name"),
			Unit:        get("unit"),
			Category:    get("category"),
			Image:       get("image"),
		}

		if row.SKU == "" {
			rowErrors = append(rowErrors, RowError{Line: line, Message: "sku kosong"})
			continue
		}
		if row.ProductName == "" {
			rowErrors = append(rowErrors, RowError{Line: line, Message: "product_name kosong"})
			continue
		}

		var parseErr error
		if row.PurchasePrice, parseErr = parseFloat(get("purchase_price")); parseErr != nil {
			rowErrors = append(rowErrors, RowError{Line: line, Message: "purchase_price tidak valid"})
			continue
		}
		if row.SellingPrice, parseErr = parseFloat(get("selling_price")); parseErr != nil {
			rowErrors = append(rowErrors, RowError{Line: line, Message: "selling_price tidak valid"})
			continue
		}
		if row.SellingPriceDebt, parseErr = parseFloat(get("selling_price_debt")); parseErr != nil {
			rowErrors = append(rowErrors, RowError{Line: line, Message: "selling_price_debt tidak valid"})
			continue
		}
		if row.Stock, parseErr = parseInt(get("stock")); parseErr != nil {
			rowErrors = append(rowErrors, RowError{Line: line, Message: "stock tidak valid"})
			continue
		}

		rows = append(rows, row)
	}

	return rows, rowErrors, nil
}

// normalizeHeader menyeragamkan nama header: huruf kecil, spasi/strip -> underscore.
func normalizeHeader(h string) string {
	h = strings.ToLower(strings.TrimSpace(h))
	h = strings.ReplaceAll(h, " ", "_")
	h = strings.ReplaceAll(h, "-", "_")
	return h
}

func isEmptyRecord(rec []string) bool {
	for _, c := range rec {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

// parseFloat: string kosong dianggap 0.
func parseFloat(s string) (float64, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.ParseFloat(s, 64)
}

// parseInt: string kosong dianggap 0.
func parseInt(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.Atoi(s)
}
