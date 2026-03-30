package enum

type MoneyPayment int

const (
	tunai MoneyPayment = iota
	hutang
	transfer
	qris
)

func (typeItem MoneyPayment) String() string {
	switch typeItem {
	case tunai:
		return "tunai"
	case hutang:
		return "hutang"
	case transfer:
		return "transfer"
	case qris:
		return "qris"
	default:
		return "unknown"
	}
}
