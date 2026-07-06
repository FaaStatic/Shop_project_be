package usecase

import "testing"

func TestApplyFee(t *testing.T) {
	cases := []struct {
		method   string
		subtotal int64
		want     int64
	}{
		{"qris", 100000, 100700}, // +0.7%
		{"qris", 10001, 10072},   // 70.007 -> ceil 71 -> 10072
		{"va", 100000, 104000},   // +Rp4.000 flat
		{"cash", 100000, 100000}, // unknown method: no fee
	}
	for _, c := range cases {
		if got := applyFee(c.method, c.subtotal); got != c.want {
			t.Fatalf("applyFee(%q,%d)=%d want %d", c.method, c.subtotal, got, c.want)
		}
	}
}
