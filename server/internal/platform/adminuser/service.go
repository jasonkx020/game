package adminuser

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/example/game/internal/platform/user"
)

type AdminUser struct {
	ID            int64      `db:"id"`
	AuthAccountID *int64     `db:"auth_account_id"`
	Phone         string     `db:"phone"`
	Nickname      string     `db:"nickname"`
	Role          string     `db:"role"`
	AvatarURL     string     `db:"avatar_url"`
	Status        string     `db:"status"`
	LastLoginAt   *time.Time `db:"last_login_at"`
}

type Service struct {
	db        *sqlx.DB
	jwtSecret []byte
	devCode   string
}

func NewService(db *sqlx.DB, jwtSecret, devCode string) *Service {
	return &Service{db: db, jwtSecret: []byte(jwtSecret), devCode: devCode}
}

// EnsureFromAuth links a local admin_users row to an auth-service account (must already exist by phone).
// phone is Auth-sourced cache (refreshed from claims); nickname/role on existing rows are not overwritten.
func (s *Service) EnsureFromAuth(ctx context.Context, accountID int64, phone, nickname, role string) (*AdminUser, error) {
	_ = nickname // Game/admin display nickname is local; claims nickname is ignored after link
	_ = role     // role stays on admin_users; Auth JWT role is checked separately
	if accountID <= 0 {
		return nil, errors.New("invalid account id")
	}
	var u AdminUser
	err := s.db.GetContext(ctx, &u,
		`SELECT id, auth_account_id, phone, nickname, role, avatar_url, status, last_login_at
		 FROM admin_users WHERE auth_account_id=$1`, accountID)
	if err == nil {
		return s.refreshAdminPhoneCache(ctx, u, phone)
	}
	if phone != "" {
		err = s.db.GetContext(ctx, &u,
			`SELECT id, auth_account_id, phone, nickname, role, avatar_url, status, last_login_at
			 FROM admin_users WHERE phone=$1 AND status='active'`, phone)
		if err == nil {
			now := time.Now()
			_, err = s.db.ExecContext(ctx,
				`UPDATE admin_users SET auth_account_id=$1, phone=$2, last_login_at=$3, updated_at=$3 WHERE id=$4`,
				accountID, phone, now, u.ID)
			if err != nil {
				return nil, err
			}
			u.AuthAccountID = &accountID
			u.Phone = phone
			u.LastLoginAt = &now
			return &u, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}
	return nil, sql.ErrNoRows
}

func (s *Service) refreshAdminPhoneCache(ctx context.Context, u AdminUser, phone string) (*AdminUser, error) {
	now := time.Now()
	if phone != "" && phone != u.Phone {
		_, err := s.db.ExecContext(ctx,
			`UPDATE admin_users SET phone=$1, last_login_at=$2, updated_at=$2 WHERE id=$3`,
			phone, now, u.ID)
		if err != nil {
			return nil, err
		}
		u.Phone = phone
	} else {
		_, _ = s.db.ExecContext(ctx,
			`UPDATE admin_users SET last_login_at=$1, updated_at=$1 WHERE id=$2`, now, u.ID)
	}
	u.LastLoginAt = &now
	return &u, nil
}

func (s *Service) GetByID(ctx context.Context, adminID int64) (*AdminUser, error) {
	var u AdminUser
	if err := s.db.GetContext(ctx, &u,
		`SELECT id, auth_account_id, phone, nickname, role, avatar_url, status, last_login_at
		 FROM admin_users WHERE id=$1`, adminID); err != nil {
		return nil, err
	}
	return &u, nil
}

type AdminUserRow struct {
	UserID      int64      `json:"user_id"`
	PhoneMasked string     `json:"phone_masked"`
	Nickname    string     `json:"nickname"`
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}

type adminUserDBRow struct {
	ID          int64      `db:"id"`
	Phone       string     `db:"phone"`
	Nickname    string     `db:"nickname"`
	Role        string     `db:"role"`
	Status      string     `db:"status"`
	CreatedAt   time.Time  `db:"created_at"`
	LastLoginAt *time.Time `db:"last_login_at"`
}

func (s *Service) ListAdmin(ctx context.Context, page, pageSize int) ([]AdminUserRow, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	var total int
	if err := s.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM admin_users`); err != nil {
		return nil, 0, err
	}

	var rows []adminUserDBRow
	err := s.db.SelectContext(ctx, &rows,
		`SELECT id, phone, nickname, role, status, created_at, last_login_at
		 FROM admin_users ORDER BY id DESC LIMIT $1 OFFSET $2`,
		pageSize, offset)
	if err != nil {
		return nil, 0, err
	}

	out := make([]AdminUserRow, len(rows))
	for i, r := range rows {
		out[i] = AdminUserRow{
			UserID:      r.ID,
			PhoneMasked: user.MaskPhone(r.Phone),
			Nickname:    r.Nickname,
			Role:        r.Role,
			Status:      r.Status,
			CreatedAt:   r.CreatedAt,
			LastLoginAt: r.LastLoginAt,
		}
	}
	return out, total, nil
}
