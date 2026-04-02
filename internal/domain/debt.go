package domain

import "time"

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

	User *Users `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
