package club

import (
	"testing"
)

func TestErrTypes(t *testing.T) {
	if ErrNotFound == nil || ErrForbidden == nil {
		t.Fatal("error constants should be defined")
	}
}
