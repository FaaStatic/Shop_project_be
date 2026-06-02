// Package pdf menyediakan generator dokumen PDF untuk laporan transaksi toko.
// File yang dihasilkan disimpan ke folder storage dan dapat diakses (di-download)
// oleh client melalui URL relatif yang dikembalikan tiap fungsi.
package pdf

import (
	"fmt"
	"strconv"
	"time"
)

// reportDir adalah folder tempat file PDF REAL disimpan (relatif terhadap root
// aplikasi). Variabel (bukan const) agar test bisa mengarahkannya ke folder
// terpisah (storage/reports_test) tanpa mencemari output produksi.
var reportDir = "storage/reports"

// urlPrefix adalah prefix URL publik untuk meng-akses file yang dihasilkan.
const urlPrefix = "/storage/reports"

// MonthReportData adalah data yang dibutuhkan untuk membuat laporan bulanan.
type MonthReportData struct {
	StoreName        string
	Cashier          string
	Month            int
	Year             int
	TotalTransaction int64   // jumlah transaksi dalam sebulan
	TotalRevenue     float64 // pendapatan masuk (selain hutang)
	TotalDebt        float64 // nilai transaksi hutang (belum jadi pendapatan)
	GrandTotal       float64 // total seluruh nilai transaksi
	Daily            []MonthReportDailyRow      // rincian per hari
	ProductsSold     []MonthReportProductRow    // rekap barang terjual sebulan
	DailyProducts    []MonthReportDailyProducts // barang terjual per hari
	GeneratedAt      time.Time
}

// MonthReportDailyRow adalah satu baris rincian transaksi pada satu hari.
type MonthReportDailyRow struct {
	Date             time.Time
	TotalTransaction int64
	Revenue          float64 // pendapatan masuk (selain hutang)
	Debt             float64 // nilai transaksi hutang
	Total            float64 // total seluruh nilai transaksi
}

// MonthReportProductRow adalah satu baris rekap produk terjual.
type MonthReportProductRow struct {
	ProductName string
	Qty         float64
	Total       float64
}

// MonthReportDailyProducts adalah daftar produk yang terjual pada satu tanggal.
type MonthReportDailyProducts struct {
	Date     time.Time
	Products []MonthReportProductRow
}

// TransactionReportItem adalah satu baris produk pada struk transaksi.
type TransactionReportItem struct {
	ProductName string
	Qty         float64
	Price       float64
	Subtotal    float64
}

// TransactionReportData adalah data yang dibutuhkan untuk membuat struk transaksi.
type TransactionReportData struct {
	StoreName   string
	NoInvoice   string
	Cashier     string
	Customer    string
	PaymentType string
	CreatedAt   time.Time
	Items       []TransactionReportItem
	Total       float64
	GeneratedAt time.Time
}

// GenerateMonthReport membuat PDF laporan bulanan berisi total transaksi dan
// pendapatan selama satu bulan, lalu mengembalikan URL relatif file tersebut.
func GenerateMonthReport(data MonthReportData) (string, error) {
	pdf := newDocument()
	pdf.AddPage()

	storeName := data.StoreName
	if storeName == "" {
		storeName = "Shop Project"
	}

	// Header
	pdf.SetFont("Arial", "B", 18)
	pdf.CellFormat(0, 10, storeName, "", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "B", 13)
	pdf.CellFormat(0, 8, "Laporan Bulanan Transaksi", "", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "", 11)
	pdf.CellFormat(0, 7, fmt.Sprintf("Periode: %s %d", monthNameID(data.Month), data.Year), "", 1, "C", false, 0, "")
	pdf.Ln(2)
	drawLine(pdf)
	pdf.Ln(4)

	// Info meta
	pdf.SetFont("Arial", "", 10)
	if data.Cashier != "" {
		pdf.CellFormat(0, 6, "Dibuat oleh : "+data.Cashier, "", 1, "L", false, 0, "")
	}
	pdf.CellFormat(0, 6, "Dicetak pada : "+formatDateTime(data.GeneratedAt), "", 1, "L", false, 0, "")
	pdf.Ln(4)

	// Ringkasan
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(0, 8, "Ringkasan", "", 1, "L", false, 0, "")
	pdf.Ln(1)

	rows := [][2]string{
		{"Jumlah Transaksi", strconv.FormatInt(data.TotalTransaction, 10) + " transaksi"},
		{"Total Pendapatan (Lunas)", formatRupiah(data.TotalRevenue)},
		{"Total Hutang (Belum Lunas)", formatRupiah(data.TotalDebt)},
		{"Total Nilai Transaksi", formatRupiah(data.GrandTotal)},
	}
	for i, row := range rows {
		// baris terakhir (total nilai) ditebalkan
		if i == len(rows)-1 {
			pdf.SetFont("Arial", "B", 11)
		} else {
			pdf.SetFont("Arial", "", 11)
		}
		pdf.CellFormat(110, 8, row[0], "1", 0, "L", false, 0, "")
		pdf.CellFormat(70, 8, row[1], "1", 1, "R", false, 0, "")
	}

	pdf.Ln(8)

	// Rincian harian: tiap tanggal yang ada transaksi.
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(0, 8, "Rincian Harian", "", 1, "L", false, 0, "")
	pdf.Ln(1)

	drawDailyHeader(pdf)
	if len(data.Daily) == 0 {
		pdf.SetFont("Arial", "I", 10)
		pdf.CellFormat(180, 8, "Tidak ada transaksi pada bulan ini.", "1", 1, "C", false, 0, "")
	} else {
		pdf.SetFont("Arial", "", 9)
		for _, d := range data.Daily {
			// Ulang header bila baris baru akan melewati batas bawah halaman.
			if pdf.GetY() > 270 {
				pdf.AddPage()
				drawDailyHeader(pdf)
				pdf.SetFont("Arial", "", 9)
			}
			pdf.CellFormat(30, 7, formatDateOnly(d.Date), "1", 0, "L", false, 0, "")
			pdf.CellFormat(25, 7, strconv.FormatInt(d.TotalTransaction, 10), "1", 0, "C", false, 0, "")
			pdf.CellFormat(45, 7, formatRupiah(d.Revenue), "1", 0, "R", false, 0, "")
			pdf.CellFormat(35, 7, formatRupiah(d.Debt), "1", 0, "R", false, 0, "")
			pdf.CellFormat(45, 7, formatRupiah(d.Total), "1", 1, "R", false, 0, "")
		}
		// Baris total keseluruhan.
		pdf.SetFont("Arial", "B", 9)
		pdf.CellFormat(30, 8, "TOTAL", "1", 0, "L", false, 0, "")
		pdf.CellFormat(25, 8, strconv.FormatInt(data.TotalTransaction, 10), "1", 0, "C", false, 0, "")
		pdf.CellFormat(45, 8, formatRupiah(data.TotalRevenue), "1", 0, "R", false, 0, "")
		pdf.CellFormat(35, 8, formatRupiah(data.TotalDebt), "1", 0, "R", false, 0, "")
		pdf.CellFormat(45, 8, formatRupiah(data.GrandTotal), "1", 1, "R", false, 0, "")
	}

	pdf.Ln(8)

	// Rekap barang terjual selama sebulan (agregat per produk).
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(0, 8, "Daftar Barang Terjual (Rekap Bulan)", "", 1, "L", false, 0, "")
	pdf.Ln(1)

	drawProductHeader(pdf)
	if len(data.ProductsSold) == 0 {
		pdf.SetFont("Arial", "I", 10)
		pdf.CellFormat(180, 8, "Tidak ada barang terjual pada bulan ini.", "1", 1, "C", false, 0, "")
	} else {
		pdf.SetFont("Arial", "", 9)
		var totalQty float64
		for _, p := range data.ProductsSold {
			if pdf.GetY() > 270 {
				pdf.AddPage()
				drawProductHeader(pdf)
				pdf.SetFont("Arial", "", 9)
			}
			pdf.CellFormat(100, 7, truncate(p.ProductName, 55), "1", 0, "L", false, 0, "")
			pdf.CellFormat(35, 7, formatQty(p.Qty), "1", 0, "C", false, 0, "")
			pdf.CellFormat(45, 7, formatRupiah(p.Total), "1", 1, "R", false, 0, "")
			totalQty += p.Qty
		}
		pdf.SetFont("Arial", "B", 9)
		pdf.CellFormat(100, 8, "TOTAL", "1", 0, "L", false, 0, "")
		pdf.CellFormat(35, 8, formatQty(totalQty), "1", 0, "C", false, 0, "")
		pdf.CellFormat(45, 8, formatRupiah(data.TotalRevenue+data.TotalDebt), "1", 1, "R", false, 0, "")
	}

	pdf.Ln(8)

	// Rincian barang terjual per hari (produk yang terjual di tiap tanggal).
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(0, 8, "Rincian Barang Terjual per Hari", "", 1, "L", false, 0, "")
	pdf.Ln(1)

	if len(data.DailyProducts) == 0 {
		pdf.SetFont("Arial", "I", 10)
		pdf.CellFormat(0, 6, "Tidak ada barang terjual pada bulan ini.", "", 1, "L", false, 0, "")
	} else {
		for _, day := range data.DailyProducts {
			if pdf.GetY() > 260 {
				pdf.AddPage()
			}
			// Sub-header tanggal.
			pdf.SetFont("Arial", "B", 10)
			pdf.SetFillColor(245, 245, 245)
			pdf.CellFormat(180, 7, formatDateOnly(day.Date), "1", 1, "L", true, 0, "")
			drawProductHeader(pdf)
			pdf.SetFont("Arial", "", 9)
			for _, p := range day.Products {
				if pdf.GetY() > 270 {
					pdf.AddPage()
					drawProductHeader(pdf)
					pdf.SetFont("Arial", "", 9)
				}
				pdf.CellFormat(100, 7, truncate(p.ProductName, 55), "1", 0, "L", false, 0, "")
				pdf.CellFormat(35, 7, formatQty(p.Qty), "1", 0, "C", false, 0, "")
				pdf.CellFormat(45, 7, formatRupiah(p.Total), "1", 1, "R", false, 0, "")
			}
			pdf.Ln(3)
		}
	}

	pdf.Ln(8)
	pdf.SetFont("Arial", "I", 9)
	pdf.CellFormat(0, 5, "Dokumen ini dibuat otomatis oleh sistem.", "", 1, "C", false, 0, "")

	filename := fmt.Sprintf("laporan-bulanan-%04d-%02d.pdf", data.Year, data.Month)
	return saveDocument(pdf, filename)
}

// GenerateTransactionReport membuat PDF struk transaksi yang dapat diberikan ke
// customer, lalu mengembalikan URL relatif file tersebut.
func GenerateTransactionReport(data TransactionReportData) (string, error) {
	pdf := newDocument()
	pdf.AddPage()

	storeName := data.StoreName
	if storeName == "" {
		storeName = "Shop Project"
	}

	// Header
	pdf.SetFont("Arial", "B", 18)
	pdf.CellFormat(0, 10, storeName, "", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "B", 13)
	pdf.CellFormat(0, 8, "Struk Transaksi", "", 1, "C", false, 0, "")
	pdf.Ln(2)
	drawLine(pdf)
	pdf.Ln(4)

	// Info transaksi
	pdf.SetFont("Arial", "", 10)
	metaInfo := [][2]string{
		{"No. Invoice", data.NoInvoice},
		{"Tanggal", formatDateTime(data.CreatedAt)},
		{"Kasir", data.Cashier},
		{"Pembayaran", titleCase(data.PaymentType)},
	}
	if data.Customer != "" {
		metaInfo = append(metaInfo, [2]string{"Customer", data.Customer})
	}
	for _, m := range metaInfo {
		if m[1] == "" {
			continue
		}
		pdf.CellFormat(40, 6, m[0], "", 0, "L", false, 0, "")
		pdf.CellFormat(0, 6, ": "+m[1], "", 1, "L", false, 0, "")
	}
	pdf.Ln(3)

	// Header tabel produk
	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(230, 230, 230)
	pdf.CellFormat(70, 8, "Produk", "1", 0, "L", true, 0, "")
	pdf.CellFormat(25, 8, "Qty", "1", 0, "C", true, 0, "")
	pdf.CellFormat(40, 8, "Harga", "1", 0, "R", true, 0, "")
	pdf.CellFormat(45, 8, "Subtotal", "1", 1, "R", true, 0, "")

	// Baris produk
	pdf.SetFont("Arial", "", 10)
	for _, item := range data.Items {
		pdf.CellFormat(70, 8, truncate(item.ProductName, 40), "1", 0, "L", false, 0, "")
		pdf.CellFormat(25, 8, formatQty(item.Qty), "1", 0, "C", false, 0, "")
		pdf.CellFormat(40, 8, formatRupiah(item.Price), "1", 0, "R", false, 0, "")
		pdf.CellFormat(45, 8, formatRupiah(item.Subtotal), "1", 1, "R", false, 0, "")
	}

	// Total
	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(135, 9, "TOTAL", "1", 0, "R", false, 0, "")
	pdf.CellFormat(45, 9, formatRupiah(data.Total), "1", 1, "R", false, 0, "")

	pdf.Ln(12)
	pdf.SetFont("Arial", "I", 10)
	pdf.CellFormat(0, 5, "Terima kasih atas kunjungan Anda.", "", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "I", 8)
	pdf.CellFormat(0, 5, "Dicetak pada "+formatDateTime(data.GeneratedAt), "", 1, "C", false, 0, "")

	filename := fmt.Sprintf("struk-%s.pdf", sanitizeFilename(data.NoInvoice))
	return saveDocument(pdf, filename)
}
