package domain

import (
	"time"

	"gorm.io/gorm"
)

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
