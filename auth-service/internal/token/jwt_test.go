package token_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
	"time"

	"github.com/example/auth-service/internal/token"
)

func testManager(t *testing.T, ttl time.Duration) *token.Manager {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return token.NewManager(key, "test-1", ttl)
}

func TestIssueAndParseES256(t *testing.T) {
	mgr := testManager(t, time.Hour)
	raw, expIn, err := mgr.IssueAccess(42, token.RolePlayer, token.PrincipalPlayer, "13800000001", "n")
	if err != nil {
		t.Fatal(err)
	}
	if expIn != 3600 {
		t.Fatalf("expires_in=%d", expIn)
	}
	c, err := mgr.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if c.AccountID != 42 || c.Role != token.RolePlayer {
		t.Fatalf("%+v", c)
	}
}

func TestRejectHS256(t *testing.T) {
	mgr := testManager(t, time.Hour)
	// Hand-crafted HS256-looking token won't verify as ES256
	if _, err := mgr.Parse("eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOjF9.sig"); err == nil {
		t.Fatal("expected reject")
	}
}
