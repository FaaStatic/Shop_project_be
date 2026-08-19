package enum

import (
	"errors"
	"strings"
)

type MoneyPayment int

const (
	tunai MoneyPayment = iota
	hutang
	transfer
	qris
)

// Hutang is the value of the "hutang" (credit/debt) payment type. It is
// exported so repository report SQL can reference the enum value without
// hardcoding its numeric value.
const Hutang MoneyPayment = hutang

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

func ParseMoneyPayment(moneyPaymentStr string) (MoneyPayment, error) {
	switch strings.ToLower(moneyPaymentStr) {
	case "tunai":
		return tunai, nil
	case "hutang":
		return hutang, nil
	case "transfer":
		return transfer, nil
	case "qris":
		return qris, nil
	default:
		return 0, errors.New("type payment not valid")
	}
}
