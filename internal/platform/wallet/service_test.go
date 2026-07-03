package wallet

import (
	"testing"
)

func TestProducts(t *testing.T) {
	p, ok := Products["rc_10"]
	if !ok || p.Cards != 10 || p.AmountCNY != 6 {
		t.Fatal("rc_10 product mismatch")
	}
	p50, ok := Products["rc_50"]
	if !ok || p50.Cards != 50 {
		t.Fatal("rc_50 product mismatch")
	}
}
