package refreshtoken

import "testing"

func TestOpaqueAndHash(t *testing.T) {
	a, err := NewOpaque()
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewOpaque()
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatal("opaque tokens should differ")
	}
	if Hash(a) == Hash(b) {
		t.Fatal("hashes should differ")
	}
	if Hash(a) != Hash(a) {
		t.Fatal("hash must be stable")
	}
	if len(Fingerprint(a)) != 12 {
		t.Fatalf("fingerprint len=%d", len(Fingerprint(a)))
	}
}
