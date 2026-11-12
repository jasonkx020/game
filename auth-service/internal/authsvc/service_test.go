package authsvc

import (
	"testing"
	"time"

	"github.com/example/auth-service/internal/refreshtoken"
	"github.com/example/auth-service/internal/token"
)

func TestParseAllowExpiredAndSubjectBind(t *testing.T) {
	mgr := token.NewManager("test-secret-at-least-32-bytes!!", 1*time.Second)
	access, _, err := mgr.IssueAccess(42, token.RolePlayer, token.PrincipalPlayer, "13900000042", "测试")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1100 * time.Millisecond)

	claims, err := mgr.Parse(access)
	if err == nil {
		t.Fatal("expected Parse to fail on expired token")
	}
	_ = claims

	claims, err = mgr.ParseAllowExpired(access)
	if err != nil {
		t.Fatalf("ParseAllowExpired: %v", err)
	}
	if claims.AccountID != 42 {
		t.Fatalf("sub=%d", claims.AccountID)
	}
	if claims.PrincipalType != token.PrincipalPlayer {
		t.Fatalf("typ=%s", claims.PrincipalType)
	}

	// Tampered token must fail.
	tampered := access[:len(access)-4] + "xxxx"
	if _, err := mgr.ParseAllowExpired(tampered); err == nil {
		t.Fatal("expected tampered token to fail")
	}

	raw, err := refreshtoken.NewOpaque()
	if err != nil {
		t.Fatal(err)
	}
	h1 := refreshtoken.Hash(raw)
	h2 := refreshtoken.Hash(raw)
	if h1 != h2 {
		t.Fatal("hash not stable")
	}
	if refreshtoken.Hash(raw+"x") == h1 {
		t.Fatal("hash collision unexpected")
	}
}

func TestNormalizePhone(t *testing.T) {
	if got := normalizePhone("138-0000-1111"); got != "13800001111" {
		t.Fatalf("got %q", got)
	}
}
