package adminuser

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jmoiron/sqlx"

	"github.com/example/game/internal/platform/user"
)

type AdminUser struct {
	ID          int64      `db:"id"`
	Phone       string     `db:"phone"`
	Nickname    string     `db:"nickname"`
	Role        string     `db:"role"`
	AvatarURL   string     `db:"avatar_url"`
	Status      string     `db:"status"`
	LastLoginAt *time.Time `db:"last_login_at"`
}

type Service struct {
	db        *sqlx.DB
	jwtSecret []byte
	devCode   string
}

func NewService(db *sqlx.DB, jwtSecret, devCode string) *Service {
	return &Service{db: db, jwtSecret: []byte(jwtSecret), devCode: devCode}
}

func (s *Service) Login(ctx context.Context, phone, smsCode string) (*AdminUser, string, error) {
	if smsCode != s.devCode {
		return nil, "", user.ErrInvalidSMS
	}
	var u AdminUser
	err := s.db.GetContext(ctx, &u,
		`SELECT id, phone, nickname, role, avatar_url, status, last_login_at
		 FROM admin_users WHERE phone=$1 AND status='active'`, phone)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", user.ErrInvalidSMS
	}
	if err != nil {
		return nil, "", err
	}
	_, _ = s.db.ExecContext(ctx,
		`UPDATE admin_users SET last_login_at=$1, updated_at=$1 WHERE id=$2`, time.Now(), u.ID)

	token, err := s.issueToken(u.ID, u.Role)
	if err != nil {
		return nil, "", err
	}
	return &u, token, nil
}

func (s *Service) Refresh(ctx context.Context, adminID int64) (string, error) {
	u, err := s.GetByID(ctx, adminID)
	if err != nil {
		return "", err
	}
	return s.issueToken(u.ID, u.Role)
}

func (s *Service) GetByID(ctx context.Context, adminID int64) (*AdminUser, error) {
	var u AdminUser
	if err := s.db.GetContext(ctx, &u,
		`SELECT id, phone, nickname, role, avatar_url, status, last_login_at
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

func (s *Service) issueToken(adminID int64, role string) (string, error) {
	claims := jwt.MapClaims{
		"sub":  adminID,
		"role": role,
		"typ":  user.PrincipalTypeAdmin,
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
		"iat":  time.Now().Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(s.jwtSecret)
}
