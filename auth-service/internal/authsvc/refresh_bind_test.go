package authsvc_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/example/auth-service/internal/authsvc"
	"github.com/example/auth-service/internal/refreshtoken"
	"github.com/example/auth-service/internal/store"
	"github.com/example/auth-service/internal/token"
)

func testES256Manager(t *testing.T, ttl time.Duration) *token.Manager {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return token.NewManager(key, "test", ttl)
}

// memory harness for dual-token rules without a live DB.
type memStore struct {
	accounts map[int64]*store.Account
	byPhone  map[string]*store.Account
	refresh  map[string]*store.RefreshRecord // hash -> record
	byAcc    map[int64]string                // accountID -> hash
	nextID   int64
}

func newMem() *memStore {
	return &memStore{
		accounts: map[int64]*store.Account{},
		byPhone:  map[string]*store.Account{},
		refresh:  map[string]*store.RefreshRecord{},
		byAcc:    map[int64]string{},
		nextID:   1,
	}
}

// StoreAdapter wraps memStore methods used via *store.Store — we test Manager + binding logic
// through a thin fake service helper below.

type fakeRepo struct {
	m *memStore
}

func (f *fakeRepo) getAccount(id int64) (*store.Account, error) {
	a, ok := f.m.accounts[id]
	if !ok {
		return nil, store.ErrNotFound
	}
	cp := *a
	return &cp, nil
}

func (f *fakeRepo) upsertRefresh(accountID int64, hash string, exp, graceUntil time.Time) {
	rec := &store.RefreshRecord{AccountID: accountID, TokenHash: hash, ExpiresAt: exp}
	if oldHash, ok := f.m.byAcc[accountID]; ok {
		prev := oldHash
		until := graceUntil
		rec.PrevTokenHash = &prev
		rec.PrevExpiresAt = &until
		if old, ok := f.m.refresh[oldHash]; ok {
			delete(f.m.refresh, oldHash)
			_ = old
		}
		// also drop previous prev pointer keys that only lived as current
	}
	f.m.refresh[hash] = rec
	if rec.PrevTokenHash != nil {
		// allow lookup by prev during grace
		f.m.refresh[*rec.PrevTokenHash] = rec
	}
	f.m.byAcc[accountID] = hash
}

func (f *fakeRepo) getRefresh(hash string) (*store.RefreshRecord, error) {
	r, ok := f.m.refresh[hash]
	if !ok {
		return nil, store.ErrNotFound
	}
	// prev grace check when lookup key is prev
	if r.TokenHash != hash {
		if r.PrevTokenHash == nil || *r.PrevTokenHash != hash || r.PrevExpiresAt == nil || time.Now().After(*r.PrevExpiresAt) {
			return nil, store.ErrNotFound
		}
	}
	cp := *r
	return &cp, nil
}

func dualRefresh(
	tokens *token.Manager,
	repo *fakeRepo,
	accessToken, refreshToken, expectedPrincipal string,
) (*authsvc.TokenPair, error) {
	if accessToken == "" || refreshToken == "" {
		return nil, authsvc.ErrBadRequest
	}
	claims, err := tokens.ParseAllowExpired(accessToken)
	if err != nil {
		return nil, errors.Join(authsvc.ErrUnauthorized, err)
	}
	if expectedPrincipal != "" && claims.PrincipalType != expectedPrincipal {
		return nil, authsvc.ErrUnauthorized
	}
	hash := refreshtoken.Hash(refreshToken)
	rec, err := repo.getRefresh(hash)
	if err != nil {
		return nil, authsvc.ErrUnauthorized
	}
	if time.Now().After(rec.ExpiresAt) {
		return nil, authsvc.ErrUnauthorized
	}
	if rec.AccountID != claims.AccountID {
		return nil, authsvc.ErrUnauthorized
	}
	acc, err := repo.getAccount(rec.AccountID)
	if err != nil || acc.Status != "active" {
		return nil, authsvc.ErrUnauthorized
	}
	access, expIn, err := tokens.IssueAccess(acc.ID, acc.Role, acc.PrincipalType, acc.PhoneStr(), acc.Nickname)
	if err != nil {
		return nil, err
	}
	raw, err := refreshtoken.NewOpaque()
	if err != nil {
		return nil, err
	}
	repo.upsertRefresh(acc.ID, refreshtoken.Hash(raw), time.Now().Add(90*24*time.Hour), time.Now().Add(2*time.Minute))
	return &authsvc.TokenPair{
		AccessToken: access, RefreshToken: raw, ExpiresIn: expIn, AccountID: acc.ID, Role: acc.Role,
	}, nil
}

func setup(t *testing.T) (*token.Manager, *fakeRepo, string, string) {
	t.Helper()
	mgr := testES256Manager(t, 2*time.Hour)
	m := newMem()
	phone := "13900000007"
	acc := &store.Account{ID: 7, PrincipalType: token.PrincipalPlayer, Phone: &phone, Role: token.RolePlayer, Status: "active"}
	m.accounts[7] = acc
	repo := &fakeRepo{m: m}
	access, _, _ := mgr.IssueAccess(7, token.RolePlayer, token.PrincipalPlayer, "13900000007", "p7")
	raw, _ := refreshtoken.NewOpaque()
	repo.upsertRefresh(7, refreshtoken.Hash(raw), time.Now().Add(24*time.Hour), time.Now().Add(2*time.Minute))
	return mgr, repo, access, raw
}

func TestDualRefreshSuccess(t *testing.T) {
	mgr, repo, access, refresh := setup(t)
	pair, err := dualRefresh(mgr, repo, access, refresh, token.PrincipalPlayer)
	if err != nil {
		t.Fatal(err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("empty pair")
	}
	// old refresh remains resolvable during grace (retry / 未落盘)
	if _, err := repo.getRefresh(refreshtoken.Hash(refresh)); err != nil {
		t.Fatal("old refresh should remain valid during grace")
	}
}

func TestDualRefreshGraceAllowsRetryWithOldToken(t *testing.T) {
	mgr, repo, access, refresh := setup(t)
	pair1, err := dualRefresh(mgr, repo, access, refresh, token.PrincipalPlayer)
	if err != nil {
		t.Fatal(err)
	}
	// simulate client never persisted pair1; retry with same old refresh
	pair2, err := dualRefresh(mgr, repo, access, refresh, token.PrincipalPlayer)
	if err != nil {
		t.Fatalf("grace retry failed: %v", err)
	}
	if pair2.RefreshToken == "" || pair2.RefreshToken == refresh {
		t.Fatal("expected a new refresh token on grace retry")
	}
	if pair2.RefreshToken == pair1.RefreshToken {
		// allowed but unlikely; ensure we at least got a usable pair
		t.Log("same refresh as first rotate (acceptable if deterministic)")
	}
}

func TestDualRefreshRejectsMissingAccess(t *testing.T) {
	mgr, repo, _, refresh := setup(t)
	if _, err := dualRefresh(mgr, repo, "", refresh, token.PrincipalPlayer); !errors.Is(err, authsvc.ErrBadRequest) {
		t.Fatalf("got %v", err)
	}
}

func TestDualRefreshRejectsMissingRefresh(t *testing.T) {
	mgr, repo, access, _ := setup(t)
	if _, err := dualRefresh(mgr, repo, access, "", token.PrincipalPlayer); !errors.Is(err, authsvc.ErrBadRequest) {
		t.Fatalf("got %v", err)
	}
}

func TestDualRefreshRejectsTamperedAccess(t *testing.T) {
	mgr, repo, access, refresh := setup(t)
	tampered := access[:len(access)-6] + "AAAAAA"
	if _, err := dualRefresh(mgr, repo, tampered, refresh, token.PrincipalPlayer); err == nil {
		t.Fatal("expected reject")
	}
}

func TestDualRefreshRejectsSubjectMismatch(t *testing.T) {
	mgr, repo, _, refresh := setup(t)
	otherAccess, _, _ := mgr.IssueAccess(99, token.RolePlayer, token.PrincipalPlayer, "13900000099", "p99")
	if _, err := dualRefresh(mgr, repo, otherAccess, refresh, token.PrincipalPlayer); !errors.Is(err, authsvc.ErrUnauthorized) {
		t.Fatalf("got %v", err)
	}
}

func TestDualRefreshRejectsExpiredRefresh(t *testing.T) {
	mgr, repo, access, _ := setup(t)
	raw, _ := refreshtoken.NewOpaque()
	repo.upsertRefresh(7, refreshtoken.Hash(raw), time.Now().Add(-time.Minute), time.Now().Add(-time.Minute))
	if _, err := dualRefresh(mgr, repo, access, raw, token.PrincipalPlayer); !errors.Is(err, authsvc.ErrUnauthorized) {
		t.Fatalf("got %v", err)
	}
}

func TestDualRefreshAllowsExpiredAccess(t *testing.T) {
	mgr := testES256Manager(t, 1*time.Second)
	m := newMem()
	m.accounts[7] = &store.Account{ID: 7, PrincipalType: token.PrincipalPlayer, Role: token.RolePlayer, Status: "active"}
	repo := &fakeRepo{m: m}
	access, _, _ := mgr.IssueAccess(7, token.RolePlayer, token.PrincipalPlayer, "13900000007", "p7")
	time.Sleep(1100 * time.Millisecond)
	raw, _ := refreshtoken.NewOpaque()
	repo.upsertRefresh(7, refreshtoken.Hash(raw), time.Now().Add(time.Hour), time.Now().Add(2*time.Minute))
	pair, err := dualRefresh(mgr, repo, access, raw, token.PrincipalPlayer)
	if err != nil {
		t.Fatal(err)
	}
	if pair.AccountID != 7 {
		t.Fatalf("id=%d", pair.AccountID)
	}
	_ = context.Background()
}
