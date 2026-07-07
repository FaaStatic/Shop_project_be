package pdf

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
)

// newDocument creates an A4 PDF document with standard margins.
func newDocument() *fpdf.Fpdf {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 15)
	return pdf
}

// saveDocument writes the document to the storage folder and returns its relative URL.
func saveDocument(pdf *fpdf.Fpdf, filename string) (string, error) {
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create report folder: %w", err)
	}
	fullPath := filepath.Join(reportDir, filename)
	if err := pdf.OutputFileAndClose(fullPath); err != nil {
		return "", fmt.Errorf("failed to save pdf file: %w", err)
	}
	return urlPrefix + "/" + filename, nil
}

// drawLine menggambar garis horizontal selebar area konten.
func drawLine(pdf *fpdf.Fpdf) {
	x := pdf.GetX()
	y := pdf.GetY()
	pdf.Line(x, y, 195, y)
}

// drawDailyHeader draws the header row of the daily breakdown table.
func drawDailyHeader(pdf *fpdf.Fpdf) {
	pdf.SetFont("Arial", "B", 9)
	pdf.SetFillColor(230, 230, 230)
	pdf.CellFormat(30, 8, "Tanggal", "1", 0, "L", true, 0, "")
	pdf.CellFormat(25, 8, "Transaksi", "1", 0, "C", true, 0, "")
	pdf.CellFormat(45, 8, "Pendapatan", "1", 0, "R", true, 0, "")
	pdf.CellFormat(35, 8, "Hutang", "1", 0, "R", true, 0, "")
	pdf.CellFormat(45, 8, "Total", "1", 1, "R", true, 0, "")
}

// formatRupiah formats a number into a Rupiah currency string, e.g. "Rp 1.250.000".
func formatRupiah(v float64) string {
	n := int64(math.Round(v))
	negative := n < 0
	if negative {
		n = -n
	}
	digits := strconv.FormatInt(n, 10)

	var b strings.Builder
	for i, d := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(d)
	}
	result := "Rp " + b.String()
	if negative {
		result = "-" + result
	}
	return result
}

// formatQty formats qty: no decimals if whole, otherwise up to 2 decimals.
func formatQty(v float64) string {
	if v == math.Trunc(v) {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'f', 2, 64)
}

// formatDateTime formats time to the Indonesian format "02 Jan 2006 15:04".
func formatDateTime(t time.Time) string {
	if t.IsZero() {
		t = time.Now()
	}
	return fmt.Sprintf("%02d %s %d %02d:%02d",
		t.Day(), monthNameShortID(int(t.Month())), t.Year(), t.Hour(), t.Minute())
}

// drawProductHeader draws the header row of the products-sold table.
func drawProductHeader(pdf *fpdf.Fpdf) {
	pdf.SetFont("Arial", "B", 9)
	pdf.SetFillColor(230, 230, 230)
	pdf.CellFormat(100, 8, "Produk", "1", 0, "L", true, 0, "")
	pdf.CellFormat(35, 8, "Qty", "1", 0, "C", true, 0, "")
	pdf.CellFormat(45, 8, "Total", "1", 1, "R", true, 0, "")
}

// drawDebtPaymentHeader draws the header row of the debt payment history table.
func drawDebtPaymentHeader(pdf *fpdf.Fpdf) {
	pdf.SetFont("Arial", "B", 9)
	pdf.SetFillColor(230, 230, 230)
	pdf.CellFormat(50, 8, "Tanggal", "1", 0, "L", true, 0, "")
	pdf.CellFormat(80, 8, "Kasir", "1", 0, "L", true, 0, "")
	pdf.CellFormat(50, 8, "Nominal", "1", 1, "R", true, 0, "")
}

// formatDateOnly formats a date to "02 Jan 2006".
func formatDateOnly(t time.Time) string {
	return fmt.Sprintf("%02d %s %d", t.Day(), monthNameShortID(int(t.Month())), t.Year())
}

// truncate cuts a string longer than max runes and appends an ellipsis. It
// counts and slices by rune (not byte) so multi-byte UTF-8 names are never split
// mid-character, which would emit invalid text into the PDF. For ASCII input the
// result is identical to a byte-based cut.
func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 3 {
		return string(r[:max])
	}
	return string(r[:max-3]) + "..."
}

// sanitizeFilename replaces characters that are unsafe for a file name.
func sanitizeFilename(s string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", " ", "_", ":", "-")
	cleaned := replacer.Replace(strings.TrimSpace(s))
	if cleaned == "" {
		return "transaksi"
	}
	return cleaned
}

var monthNamesID = [...]string{
	"Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember",
}

var monthNamesShortID = [...]string{
	"Jan", "Feb", "Mar", "Apr", "Mei", "Jun",
	"Jul", "Agu", "Sep", "Okt", "Nov", "Des",
}

// monthNameID returns the Indonesian month name (1-12); otherwise an empty string.
func monthNameID(month int) string {
	if month < 1 || month > 12 {
		return ""
	}
	return monthNamesID[month-1]
}

func monthNameShortID(month int) string {
	if month < 1 || month > 12 {
		return ""
	}
	return monthNamesShortID[month-1]
}

// titleCase capitalizes the first letter, e.g. "tunai" -> "Tunai".
func titleCase(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
