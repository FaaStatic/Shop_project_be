package responsedto

type CustomerDtoResponse struct {
	ID     uint              `gorm:"primaryKey" json:"id"`
	Nama   string            `gorm:"type:varchar(150);not null" json:"nama"`
	NoHP   string            `gorm:"type:varchar(15)" json:"no_hp"`
	Alamat string            `gorm:"type:text" json:"alamat"`
	Debts  []DebtResponseDto `json:"debts,omitempty"`
}
