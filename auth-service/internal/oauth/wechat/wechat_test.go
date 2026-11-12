package wechat

import (
	"context"
	"testing"
)

func TestMockStableOpenID(t *testing.T) {
	r := NewResolver(Config{Mode: "mock"})
	a, err := r.Resolve(context.Background(), PlatformMiniProgram, "abc")
	if err != nil {
		t.Fatal(err)
	}
	b, err := r.Resolve(context.Background(), PlatformMiniProgram, "abc")
	if err != nil {
		t.Fatal(err)
	}
	if a.OpenID != b.OpenID || a.UnionID != b.UnionID {
		t.Fatalf("unstable mock ids")
	}
	c, _ := r.Resolve(context.Background(), PlatformWeb, "abc")
	if c.OpenID == a.OpenID {
		t.Fatal("web and mp should differ for same code")
	}
}
