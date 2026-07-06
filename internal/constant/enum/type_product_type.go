package enum

import (
	"errors"
	"strings"
)

// ProductType distinguishes physical inventory from digital goods
// (e-wallet top-up, pulsa, data packages). Digital products are not
// stock-managed and require a destination (phone/account) at sale time.
type ProductType int

const (
	// Physical is stock-managed inventory (default).
	Physical ProductType = iota
	// Digital is a non-stock good fulfilled to a destination number/account.
	Digital
)

func (p ProductType) String() string {
	switch p {
	case Physical:
		return "physical"
	case Digital:
		return "digital"
	default:
		return "unknown"
	}
}

// IsDigital reports whether this product bypasses stock management.
func (p ProductType) IsDigital() bool { return p == Digital }

// ParseProductType accepts a number ("0"/"1") or text ("physical"/"digital").
// Empty defaults to Physical.
func ParseProductType(s string) (ProductType, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "physical", "0":
		return Physical, nil
	case "digital", "1":
		return Digital, nil
	default:
		return 0, errors.New("invalid product type")
	}
}
