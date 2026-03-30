package enum

type ProductUnit int

const (
	pcs ProductUnit = iota
	kg
	liter
	kardus
	ikat
)

func (typeItem ProductUnit) String() string {
	switch typeItem {
	case pcs:
		return "pcs"
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
