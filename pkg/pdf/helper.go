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

// newDocument membuat dokumen PDF A4 dengan margin standar.
func newDocument() *fpdf.Fpdf {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 15)
	return pdf
}

// saveDocument menulis dokumen ke folder storage dan mengembalikan URL relatifnya.
func saveDocument(pdf *fpdf.Fpdf, filename string) (string, error) {
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		return "", fmt.Errorf("gagal membuat folder laporan: %w", err)
	}
	fullPath := filepath.Join(reportDir, filename)
	if err := pdf.OutputFileAndClose(fullPath); err != nil {
		return "", fmt.Errorf("gagal menyimpan file pdf: %w", err)
	}
	return urlPrefix + "/" + filename, nil
}

// drawLine menggambar garis horizontal selebar area konten.
func drawLine(pdf *fpdf.Fpdf) {
	x := pdf.GetX()
	y := pdf.GetY()
	pdf.Line(x, y, 195, y)
}

// drawDailyHeader menggambar baris header tabel rincian harian.
func drawDailyHeader(pdf *fpdf.Fpdf) {
	pdf.SetFont("Arial", "B", 9)
	pdf.SetFillColor(230, 230, 230)
	pdf.CellFormat(30, 8, "Tanggal", "1", 0, "L", true, 0, "")
	pdf.CellFormat(25, 8, "Transaksi", "1", 0, "C", true, 0, "")
	pdf.CellFormat(45, 8, "Pendapatan", "1", 0, "R", true, 0, "")
	pdf.CellFormat(35, 8, "Hutang", "1", 0, "R", true, 0, "")
	pdf.CellFormat(45, 8, "Total", "1", 1, "R", true, 0, "")
}

// formatRupiah memformat angka menjadi string mata uang Rupiah, contoh: "Rp 1.250.000".
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

// formatQty memformat qty: tanpa desimal bila bulat, selain itu maksimal 2 desimal.
func formatQty(v float64) string {
	if v == math.Trunc(v) {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'f', 2, 64)
}

// formatDateTime memformat waktu ke format Indonesia "02 Jan 2006 15:04".
func formatDateTime(t time.Time) string {
	if t.IsZero() {
		t = time.Now()
	}
	return fmt.Sprintf("%02d %s %d %02d:%02d",
		t.Day(), monthNameShortID(int(t.Month())), t.Year(), t.Hour(), t.Minute())
}

// drawProductHeader menggambar baris header tabel produk terjual.
func drawProductHeader(pdf *fpdf.Fpdf) {
	pdf.SetFont("Arial", "B", 9)
	pdf.SetFillColor(230, 230, 230)
	pdf.CellFormat(100, 8, "Produk", "1", 0, "L", true, 0, "")
	pdf.CellFormat(35, 8, "Qty", "1", 0, "C", true, 0, "")
	pdf.CellFormat(45, 8, "Total", "1", 1, "R", true, 0, "")
}

// drawDebtPaymentHeader menggambar baris header tabel riwayat pembayaran hutang.
func drawDebtPaymentHeader(pdf *fpdf.Fpdf) {
	pdf.SetFont("Arial", "B", 9)
	pdf.SetFillColor(230, 230, 230)
	pdf.CellFormat(50, 8, "Tanggal", "1", 0, "L", true, 0, "")
	pdf.CellFormat(80, 8, "Kasir", "1", 0, "L", true, 0, "")
	pdf.CellFormat(50, 8, "Nominal", "1", 1, "R", true, 0, "")
}

// formatDateOnly memformat tanggal ke format "02 Jan 2006".
func formatDateOnly(t time.Time) string {
	return fmt.Sprintf("%02d %s %d", t.Day(), monthNameShortID(int(t.Month())), t.Year())
}

// truncate memotong string yang lebih panjang dari max dan menambahkan elipsis.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

// sanitizeFilename mengganti karakter yang tidak aman untuk nama file.
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

// monthNameID mengembalikan nama bulan Indonesia (1-12); selain itu string kosong.
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

// titleCase mengubah huruf pertama menjadi kapital, contoh: "tunai" -> "Tunai".
func titleCase(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
