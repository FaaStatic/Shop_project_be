// Package sheet reads product data from a CSV or Excel (.xlsx) file.
// This parser is generic: it does not depend on the domain/usecase layer. The caller
// maps the result to application entities.
package sheet

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// ProductRow is a single parsed product row (raw values).
type ProductRow struct {
	Line             int // row number in the file (1-based, including the header)
	SKU              string
	ProductName      string
	Unit             string // raw; parsed by the caller (number or text)
	PurchasePrice    float64
	SellingPrice     float64
	SellingPriceDebt float64
	Stock            float64
	Category         string
	Image            string
	ProductType      string // raw; parsed by the caller. Empty -> physical.
}

// RowError marks a row that failed to parse along with the reason.
type RowError struct {
	Line    int
	Message string
}

func (e RowError) Error() string {
	return fmt.Sprintf("row %d: %s", e.Line, e.Message)
}

// required columns in the file.
var requiredHeaders = []string{"sku", "product_name", "purchase_price", "selling_price", "selling_price_debt"}

// ParseProducts reads products from r. The format is determined by the filename extension
// (.csv or .xlsx). Returns valid rows, a list of per-row errors (skipped
// rows), and a fatal error (file unreadable / incomplete header).
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
		return nil, nil, fmt.Errorf("unsupported file format (use .csv or .xlsx)")
	}
}

func readCSV(r io.Reader) ([][]string, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1 // allow a varying number of columns
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read csv: %w", err)
	}
	return records, nil
}

func readXLSX(r io.Reader) ([][]string, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read excel: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("excel file has no sheet")
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("failed to read excel rows: %w", err)
	}
	return rows, nil
}

// parseRecords maps raw rows (records) into []ProductRow
// based on the header in the first row. The err parameter forwards a read error.
func parseRecords(records [][]string, err error) ([]ProductRow, []RowError, error) {
	if err != nil {
		return nil, nil, err
	}
	if len(records) == 0 {
		return nil, nil, fmt.Errorf("file is empty")
	}

	// Map header name -> column index.
	colIndex := make(map[string]int)
	for i, h := range records[0] {
		colIndex[normalizeHeader(h)] = i
	}
	for _, h := range requiredHeaders {
		if _, ok := colIndex[h]; !ok {
			return nil, nil, fmt.Errorf("required column '%s' not found in header", h)
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
			ProductType: get("product_type"),
		}

		if row.SKU == "" {
			rowErrors = append(rowErrors, RowError{Line: line, Message: "sku is empty"})
			continue
		}
		if row.ProductName == "" {
			rowErrors = append(rowErrors, RowError{Line: line, Message: "product_name is empty"})
			continue
		}

		var parseErr error
		if row.PurchasePrice, parseErr = parseFloat(get("purchase_price")); parseErr != nil {
			rowErrors = append(rowErrors, RowError{Line: line, Message: "invalid purchase_price"})
			continue
		}
		if row.SellingPrice, parseErr = parseFloat(get("selling_price")); parseErr != nil {
			rowErrors = append(rowErrors, RowError{Line: line, Message: "invalid selling_price"})
			continue
		}
		if row.SellingPriceDebt, parseErr = parseFloat(get("selling_price_debt")); parseErr != nil {
			rowErrors = append(rowErrors, RowError{Line: line, Message: "invalid selling_price_debt"})
			continue
		}
		if row.Stock, parseErr = parseFloat(get("stock")); parseErr != nil {
			rowErrors = append(rowErrors, RowError{Line: line, Message: "invalid stock"})
			continue
		}

		rows = append(rows, row)
	}

	return rows, rowErrors, nil
}

// normalizeHeader normalizes a header name: lowercase, space/dash -> underscore.
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

// parseFloat: an empty string is treated as 0.
func parseFloat(s string) (float64, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.ParseFloat(s, 64)
}

// parseInt: an empty string is treated as 0.
func parseInt(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.Atoi(s)
}
