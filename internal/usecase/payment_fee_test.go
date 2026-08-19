package usecase

import "testing"

func TestApplyFee(t *testing.T) {
	cases := []struct {
		method   string
		subtotal int64
		want     int64
	}{
		{"qris", 100000, 100784}, // gross = ceil(subtotal / (1 - 0.00777))
		{"qris", 10001, 10080},   // fee-on-fee, rounded UP
		{"va", 100000, 104440},   // +Rp4.440 (Rp4.000 + 11% VAT)
		{"cash", 100000, 100000}, // unknown method: no fee
	}
	for _, c := range cases {
		if got := applyFee(c.method, c.subtotal); got != c.want {
			t.Fatalf("applyFee(%q,%d)=%d want %d", c.method, c.subtotal, got, c.want)
		}
	}
}
