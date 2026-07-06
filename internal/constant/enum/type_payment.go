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
	kartu // debit/credit card payment via Midtrans
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
	case kartu:
		return "kartu"
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
	case "kartu":
		return kartu, nil
	default:
		return 0, errors.New("type payment not valid")
	}
}
