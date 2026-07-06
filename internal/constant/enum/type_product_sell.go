package enum

import (
	"errors"
	"strings"
)

type ProductUnit int

const (
	pcs ProductUnit = iota
	gram
	kg
	liter
	kardus
	ikat
)

func (typeItem ProductUnit) String() string {
	switch typeItem {
	case pcs:
		return "pcs"
	case gram:
		return "gram"
	case kg:
		return "kg"
	case liter:
		return "liter"
	case kardus:
		return "kardus"
	case ikat:
		return "ikat"
	default:
		return "unknown"
	}
}

// ParseProductUnit accepts a unit as a number (0-4) or text
// ("pcs", "kg", "liter", "kardus", "ikat"). Empty defaults to "pcs".
func ParseProductUnit(s string) (ProductUnit, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "pcs", "0":
		return pcs, nil
	case "kg", "1":
		return kg, nil
	case "liter", "2":
		return liter, nil
	case "kardus", "3":
		return kardus, nil
	case "ikat", "4":
		return ikat, nil
	default:
		return 0, errors.New("invalid unit")
	}
}
