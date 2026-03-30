package models

import (
	"time"

	"gorm.io/gorm"
)

type Users struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	Username     string `gorm:"type:varchar(100);uniqueIndex;not null" json:"username"`
	PasswordHash string `gorm:"type:varchar(255);not null" json:"-"`
	Role         string `gorm:"type:enum('superadmin','admin','staff');default:'staff'" json:"role"`

	Transactions []Transactions `gorm:"foreignKey:UserID" json:"transactions,omitempty"`
}

type Products struct {
	ID              uint    `gorm:"primaryKey" json:"id"`
	SKU             string  `gorm:"type:varchar(50);uniqueIndex" json:"sku"`
	NamaProduk      string  `gorm:"type:varchar(255);not null" json:"nama_produk"`
	Satuan          string  `gorm:"type:enum('pcs','kg','liter','kardus','ikat');not null" json:"satuan"`
	HargaBeli       float64 `gorm:"type:decimal(15,2);not null" json:"harga_beli"`
	HargaJualTunai  float64 `gorm:"type:decimal(15,2);not null" json:"harga_jual_tunai"`
	HargaJualHutang float64 `gorm:"type:decimal(15,2);not null" json:"harga_jual_hutang"`
	Stok            float64 `gorm:"type:decimal(10,2);default:0" json:"stok"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type Transactions struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	NoInvoice      string    `gorm:"type:varchar(50);uniqueIndex;not null" json:"no_invoice"`
	UserID         uint      `gorm:"not null" json:"user_id"`
	CustomerID     *uint     `json:"customer_id"`
	TipePembayaran string    `gorm:"type:enum('tunai','hutang','transfer','qris');not null" json:"tipe_pembayaran"`
	TotalTransaksi float64   `gorm:"type:decimal(15,2);not null" json:"total_transaksi"`
	TotalLaba      float64   `gorm:"type:decimal(15,2);not null" json:"total_laba"`
	CreatedAt      time.Time `json:"created_at"`

	User              Users                `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Customer          Customers            `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	TransactionDetail []TransactionsDetail `gorm:"foreignKey:TransactionID" json:"details"`
}

type TransactionsDetail struct {
	ID            uint    `gorm:"primaryKey" json:"id"`
	TransactionID uint    `gorm:"not null" json:"transaction_id"`
	ProductID     uint    `gorm:"not null" json:"product_id"`
	Qty           float64 `gorm:"type:decimal(8,2);not null" json:"qty"`
	Subtotal      float64 `gorm:"type:decimal(15,2);not null" json:"subtotal"`

	Product Products `gorm:"foreignKey:ProductID" json:"product,omitempty"`
}

type Customers struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	Nama   string `gorm:"type:varchar(150);not null" json:"nama"`
	NoHP   string `gorm:"type:varchar(15)" json:"no_hp"`
	Alamat string `gorm:"type:text" json:"alamat"`

	Transactions []Transactions `gorm:"foreignKey:CustomerID" json:"transactions,omitempty"`
	Debts        []Debts        `gorm:"foreignKey:CustomerID" json:"debts,omitempty"`
}

type Debts struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	TransactionID uint      `gorm:"not null;unique" json:"transaction_id"`
	CustomerID    uint      `gorm:"not null" json:"customer_id"`
	TotalHutang   float64   `gorm:"type:decimal(15,2);not null" json:"total_hutang"`
	SisaHutang    float64   `gorm:"type:decimal(15,2);not null" json:"sisa_hutang"`
	Status        string    `gorm:"type:enum('belum_lunas','lunas');default:'belum_lunas'" json:"status"`
	JatuhTempo    time.Time `json:"jatuh_tempo"`

	Transaction  Transactions   `gorm:"foreignKey:TransactionID" json:"transaction,omitempty"`
	Customer     Customers      `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	DebtPayments []DebtPayments `gorm:"foreignKey:DebtID" json:"payments"`
}

type DebtPayments struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	DebtID       uint      `gorm:"not null" json:"debt_id"`
	UserID       uint      `gorm:"not null" json:"user_id"`
	NominalBayar float64   `gorm:"type:decimal(15,2);not null" json:"nominal_bayar"`
	TanggalBayar time.Time `gorm:"autoCreateTime" json:"tanggal_bayar"`

	User Users `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
