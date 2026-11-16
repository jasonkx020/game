package player

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func newMockService(t *testing.T) (*Service, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return NewService(sqlx.NewDb(db, "sqlmock")), mock
}

func TestSeedPlayerNickname(t *testing.T) {
	if got := seedPlayerNickname("花名", "13800000001", 9); got != "花名" {
		t.Fatalf("got %q", got)
	}
	if got := seedPlayerNickname("", "13800000001", 9); got != "玩家0001" {
		t.Fatalf("got %q", got)
	}
	if got := seedPlayerNickname("", "", 42); got != "玩家42" {
		t.Fatalf("got %q", got)
	}
}

func TestEnsureFromAuth_RefreshesPhoneKeepsNickname(t *testing.T) {
	svc, mock := newMockService(t)
	aid := int64(7)
	rows := sqlmock.NewRows([]string{
		"id", "auth_account_id", "phone", "nickname", "avatar_url", "status", "last_login_at",
	}).AddRow(int64(3), aid, "13800000001", "大厅花名", "", "active", nil)

	mock.ExpectQuery(`SELECT id, auth_account_id, phone, nickname, avatar_url, status, last_login_at\s+FROM players WHERE auth_account_id=\$1`).
		WithArgs(aid).
		WillReturnRows(rows)
	mock.ExpectExec(`UPDATE players SET phone=\$1, updated_at=\$2 WHERE id=\$3`).
		WithArgs("13900000002", sqlmock.AnyArg(), int64(3)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO wallet_room_card`).
		WithArgs(int64(3)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`INSERT INTO wallet_game_coin`).
		WithArgs(int64(3)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	p, err := svc.EnsureFromAuth(context.Background(), aid, "13900000002", "JWT里的新昵称")
	if err != nil {
		t.Fatal(err)
	}
	if p.Phone != "13900000002" {
		t.Fatalf("phone=%q", p.Phone)
	}
	if p.Nickname != "大厅花名" {
		t.Fatalf("nickname overwritten: %q", p.Nickname)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEnsureFromAuth_ExistingSamePhoneNoUpdate(t *testing.T) {
	svc, mock := newMockService(t)
	aid := int64(7)
	rows := sqlmock.NewRows([]string{
		"id", "auth_account_id", "phone", "nickname", "avatar_url", "status", "last_login_at",
	}).AddRow(int64(3), aid, "13800000001", "大厅花名", "", "active", nil)

	mock.ExpectQuery(`SELECT id, auth_account_id, phone, nickname, avatar_url, status, last_login_at\s+FROM players WHERE auth_account_id=\$1`).
		WithArgs(aid).
		WillReturnRows(rows)
	mock.ExpectExec(`INSERT INTO wallet_room_card`).
		WithArgs(int64(3)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`INSERT INTO wallet_game_coin`).
		WithArgs(int64(3)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	p, err := svc.EnsureFromAuth(context.Background(), aid, "13800000001", "别的昵称")
	if err != nil {
		t.Fatal(err)
	}
	if p.Nickname != "大厅花名" || p.Phone != "13800000001" {
		t.Fatalf("%+v", p)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEnsureFromAuth_CreatesWithSeedNickname(t *testing.T) {
	svc, mock := newMockService(t)
	aid := int64(9)
	mock.ExpectQuery(`SELECT id, auth_account_id, phone, nickname, avatar_url, status, last_login_at\s+FROM players WHERE auth_account_id=\$1`).
		WithArgs(aid).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`SELECT id, auth_account_id, phone, nickname, avatar_url, status, last_login_at\s+FROM players WHERE phone=\$1`).
		WithArgs("13800000099").
		WillReturnError(sql.ErrNoRows)

	insertRows := sqlmock.NewRows([]string{
		"id", "auth_account_id", "phone", "nickname", "avatar_url", "status", "last_login_at",
	}).AddRow(int64(11), aid, "13800000099", "玩家0099", "", "active", nil)
	mock.ExpectQuery(`INSERT INTO players \(phone, nickname, auth_account_id\) VALUES \(\$1, \$2, \$3\)`).
		WithArgs("13800000099", "玩家0099", aid).
		WillReturnRows(insertRows)
	mock.ExpectExec(`INSERT INTO wallet_room_card`).
		WithArgs(int64(11)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO wallet_game_coin`).
		WithArgs(int64(11)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE players SET last_login_at=\$1, updated_at=\$1 WHERE id=\$2`).
		WithArgs(sqlmock.AnyArg(), int64(11)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	p, err := svc.EnsureFromAuth(context.Background(), aid, "13800000099", "")
	if err != nil {
		t.Fatal(err)
	}
	if p.Nickname != "玩家0099" || p.Phone != "13800000099" {
		t.Fatalf("%+v", p)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
