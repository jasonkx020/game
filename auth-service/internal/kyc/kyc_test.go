package kyc

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestMockVerifierMatchAndMismatch(t *testing.T) {
	id := makeValidID("110101", 1990, 1, 1, 1)
	v := NewMock()
	res, err := v.Verify(context.Background(), "张三", id)
	if err != nil {
		t.Fatalf("expected match: %v id=%s", err, id)
	}
	if !res.Matched || res.IDHash == "" {
		t.Fatalf("bad result %+v", res)
	}

	res, err = v.Verify(context.Background(), "拒绝", id)
	if !errors.Is(err, ErrMismatch) {
		t.Fatalf("expected mismatch, got %v", err)
	}
	if res == nil || res.Matched {
		t.Fatal("expected unmatched result")
	}
}

func TestMaskName(t *testing.T) {
	if got := MaskName("张三丰"); got != "张**" {
		t.Fatalf("got %q", got)
	}
}

func makeValidID(region string, y, m, d, seq int) string {
	body := fmt.Sprintf("%s%04d%02d%02d%03d", region, y, m, d, seq)
	weights := []int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}
	checks := []byte{'1', '0', 'X', '9', '8', '7', '6', '5', '4', '3', '2'}
	sum := 0
	for i := 0; i < 17; i++ {
		n := int(body[i] - '0')
		sum += n * weights[i]
	}
	return body + string(checks[sum%11])
}
