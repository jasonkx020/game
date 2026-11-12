package authsvc_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/example/auth-service/internal/authsvc"
	"github.com/example/auth-service/internal/refreshtoken"
	"github.com/example/auth-service/internal/store"
	"github.com/example/auth-service/internal/token"
)

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

func (f *fakeRepo) upsertRefresh(accountID int64, hash string, exp time.Time) {
	if old, ok := f.m.byAcc[accountID]; ok {
		delete(f.m.refresh, old)
	}
	rec := &store.RefreshRecord{AccountID: accountID, TokenHash: hash, ExpiresAt: exp}
	f.m.refresh[hash] = rec
	f.m.byAcc[accountID] = hash
}

func (f *fakeRepo) getRefresh(hash string) (*store.RefreshRecord, error) {
	r, ok := f.m.refresh[hash]
	if !ok {
		return nil, store.ErrNotFound
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
	repo.upsertRefresh(acc.ID, refreshtoken.Hash(raw), time.Now().Add(90*24*time.Hour))
	return &authsvc.TokenPair{
		AccessToken: access, RefreshToken: raw, ExpiresIn: expIn, AccountID: acc.ID, Role: acc.Role,
	}, nil
}

func setup() (*token.Manager, *fakeRepo, string, string) {
	mgr := token.NewManager("test-secret-at-least-32-bytes-long!!", 2*time.Hour)
	m := newMem()
	phone := "13900000007"
	acc := &store.Account{ID: 7, PrincipalType: token.PrincipalPlayer, Phone: &phone, Role: token.RolePlayer, Status: "active"}
	m.accounts[7] = acc
	repo := &fakeRepo{m: m}
	access, _, _ := mgr.IssueAccess(7, token.RolePlayer, token.PrincipalPlayer, "13900000007", "p7")
	raw, _ := refreshtoken.NewOpaque()
	repo.upsertRefresh(7, refreshtoken.Hash(raw), time.Now().Add(24*time.Hour))
	return mgr, repo, access, raw
}

func TestDualRefreshSuccess(t *testing.T) {
	mgr, repo, access, refresh := setup()
	pair, err := dualRefresh(mgr, repo, access, refresh, token.PrincipalPlayer)
	if err != nil {
		t.Fatal(err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("empty pair")
	}
	// old refresh must be gone
	if _, err := repo.getRefresh(refreshtoken.Hash(refresh)); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("old refresh should be rotated away")
	}
}

func TestDualRefreshRejectsMissingAccess(t *testing.T) {
	mgr, repo, _, refresh := setup()
	if _, err := dualRefresh(mgr, repo, "", refresh, token.PrincipalPlayer); !errors.Is(err, authsvc.ErrBadRequest) {
		t.Fatalf("got %v", err)
	}
}

func TestDualRefreshRejectsMissingRefresh(t *testing.T) {
	mgr, repo, access, _ := setup()
	if _, err := dualRefresh(mgr, repo, access, "", token.PrincipalPlayer); !errors.Is(err, authsvc.ErrBadRequest) {
		t.Fatalf("got %v", err)
	}
}

func TestDualRefreshRejectsTamperedAccess(t *testing.T) {
	mgr, repo, access, refresh := setup()
	tampered := access[:len(access)-6] + "AAAAAA"
	if _, err := dualRefresh(mgr, repo, tampered, refresh, token.PrincipalPlayer); err == nil {
		t.Fatal("expected reject")
	}
}

func TestDualRefreshRejectsSubjectMismatch(t *testing.T) {
	mgr, repo, _, refresh := setup()
	otherAccess, _, _ := mgr.IssueAccess(99, token.RolePlayer, token.PrincipalPlayer, "13900000099", "p99")
	if _, err := dualRefresh(mgr, repo, otherAccess, refresh, token.PrincipalPlayer); !errors.Is(err, authsvc.ErrUnauthorized) {
		t.Fatalf("got %v", err)
	}
}

func TestDualRefreshRejectsExpiredRefresh(t *testing.T) {
	mgr, repo, access, _ := setup()
	raw, _ := refreshtoken.NewOpaque()
	repo.upsertRefresh(7, refreshtoken.Hash(raw), time.Now().Add(-time.Minute))
	if _, err := dualRefresh(mgr, repo, access, raw, token.PrincipalPlayer); !errors.Is(err, authsvc.ErrUnauthorized) {
		t.Fatalf("got %v", err)
	}
}

func TestDualRefreshAllowsExpiredAccess(t *testing.T) {
	mgr := token.NewManager("test-secret-at-least-32-bytes-long!!", 1*time.Second)
	m := newMem()
	m.accounts[7] = &store.Account{ID: 7, PrincipalType: token.PrincipalPlayer, Role: token.RolePlayer, Status: "active"}
	repo := &fakeRepo{m: m}
	access, _, _ := mgr.IssueAccess(7, token.RolePlayer, token.PrincipalPlayer, "13900000007", "p7")
	time.Sleep(1100 * time.Millisecond)
	raw, _ := refreshtoken.NewOpaque()
	repo.upsertRefresh(7, refreshtoken.Hash(raw), time.Now().Add(time.Hour))
	pair, err := dualRefresh(mgr, repo, access, raw, token.PrincipalPlayer)
	if err != nil {
		t.Fatal(err)
	}
	if pair.AccountID != 7 {
		t.Fatalf("id=%d", pair.AccountID)
	}
	_ = context.Background()
}
