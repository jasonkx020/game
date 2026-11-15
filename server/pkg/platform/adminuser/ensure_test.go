package adminuser

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestEnsureFromAuth_RefreshesPhoneKeepsNickname(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	svc := NewService(sqlx.NewDb(db, "sqlmock"), "secret", "123456")

	aid := int64(5)
	rows := sqlmock.NewRows([]string{
		"id", "auth_account_id", "phone", "nickname", "role", "avatar_url", "status", "last_login_at",
	}).AddRow(int64(2), aid, "13800000000", "平台管理员", "platform_admin", "", "active", nil)

	mock.ExpectQuery(`SELECT id, auth_account_id, phone, nickname, role, avatar_url, status, last_login_at\s+FROM admin_users WHERE auth_account_id=\$1`).
		WithArgs(aid).
		WillReturnRows(rows)
	mock.ExpectExec(`UPDATE admin_users SET phone=\$1, last_login_at=\$2, updated_at=\$2 WHERE id=\$3`).
		WithArgs("13900000000", sqlmock.AnyArg(), int64(2)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	u, err := svc.EnsureFromAuth(context.Background(), aid, "13900000000", "JWT昵称", "platform_admin")
	if err != nil {
		t.Fatal(err)
	}
	if u.Phone != "13900000000" {
		t.Fatalf("phone=%q", u.Phone)
	}
	if u.Nickname != "平台管理员" {
		t.Fatalf("nickname overwritten: %q", u.Nickname)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
