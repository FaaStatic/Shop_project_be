package responsedto

import "github.com/google/uuid"

type CustomerDtoResponse struct {
	ID     uuid.UUID         `json:"id"`
	Nama   string            `json:"nama"`
	NoHP   string            `json:"no_hp"`
	Alamat string            `json:"alamat"`
	Debts  []DebtResponseDto `json:"debts,omitempty"`
}
