package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var ErrNotFound = errors.New("not found")

// IsUniqueViolation reports PostgreSQL unique_violation (23505).
func IsUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}

type Account struct {
	ID                  int64      `db:"id"`
	PrincipalType       string     `db:"principal_type"`
	Phone               *string    `db:"phone"`
	Email               *string    `db:"email"`
	Role                string     `db:"role"`
	Nickname            string     `db:"nickname"`
	AvatarURL           string     `db:"avatar_url"`
	Status              string     `db:"status"`
	RealnameStatus      string     `db:"realname_status"`
	RealName            string     `db:"real_name"`
	IDCardHash          *string    `db:"id_card_hash"`
	IDCardMasked        string     `db:"id_card_masked"`
	BirthDate           *time.Time `db:"birth_date"`
	IsMinor             bool       `db:"is_minor"`
	RealnameVerifiedAt  *time.Time `db:"realname_verified_at"`
	CreatedAt           time.Time  `db:"created_at"`
	UpdatedAt           time.Time  `db:"updated_at"`
	LastLoginAt         *time.Time `db:"last_login_at"`
}

func (a *Account) PhoneStr() string {
	if a.Phone == nil {
		return ""
	}
	return *a.Phone
}

func (a *Account) EmailStr() string {
	if a.Email == nil {
		return ""
	}
	return *a.Email
}

type RefreshRecord struct {
	AccountID     int64      `db:"account_id"`
	TokenHash     string     `db:"token_hash"`
	ExpiresAt     time.Time  `db:"expires_at"`
	PrevTokenHash *string    `db:"prev_token_hash"`
	PrevExpiresAt *time.Time `db:"prev_expires_at"`
	CreatedAt     time.Time  `db:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at"`
}

type OAuthIdentity struct {
	ID        int64     `db:"id"`
	AccountID int64     `db:"account_id"`
	Provider  string    `db:"provider"`
	Platform  string    `db:"platform"`
	OpenID    string    `db:"open_id"`
	UnionID   *string   `db:"union_id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

const accountCols = `id, principal_type, phone, email, role, nickname, avatar_url, status,
	realname_status, real_name, id_card_hash, id_card_masked, birth_date, is_minor, realname_verified_at,
	created_at, updated_at, last_login_at`

type Store struct {
	db *sqlx.DB
}

func Connect(databaseURL string) (*Store, error) {
	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) DB() *sqlx.DB { return s.db }

func (s *Store) GetAccountByPhone(ctx context.Context, phone string) (*Account, error) {
	var a Account
	err := s.db.GetContext(ctx, &a, `SELECT `+accountCols+` FROM auth_accounts WHERE phone=$1`, phone)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &a, err
}

func (s *Store) GetAccountByEmail(ctx context.Context, email string) (*Account, error) {
	var a Account
	err := s.db.GetContext(ctx, &a, `SELECT `+accountCols+` FROM auth_accounts WHERE email=$1`, email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &a, err
}

func (s *Store) GetAccountByID(ctx context.Context, id int64) (*Account, error) {
	var a Account
	err := s.db.GetContext(ctx, &a, `SELECT `+accountCols+` FROM auth_accounts WHERE id=$1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &a, err
}

type CreateAccountParams struct {
	PrincipalType string
	Phone         *string
	Email         *string
	Role          string
	Nickname      string
	AvatarURL     string
}

func (s *Store) CreateAccount(ctx context.Context, p CreateAccountParams) (*Account, error) {
	var a Account
	err := s.db.QueryRowxContext(ctx, `
		INSERT INTO auth_accounts (principal_type, phone, email, role, nickname, avatar_url)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING `+accountCols,
		p.PrincipalType, p.Phone, p.Email, p.Role, p.Nickname, p.AvatarURL).StructScan(&a)
	return &a, err
}

func (s *Store) TouchLogin(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE auth_accounts SET last_login_at=$1, updated_at=$1 WHERE id=$2`, time.Now(), id)
	return err
}

// UpdateStatus sets auth_accounts.status (active|disabled).
func (s *Store) UpdateStatus(ctx context.Context, id int64, status string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE auth_accounts SET status=$1, updated_at=$2 WHERE id=$3`, status, time.Now(), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateRole sets auth_accounts.role.
func (s *Store) UpdateRole(ctx context.Context, id int64, role string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE auth_accounts SET role=$1, updated_at=$2 WHERE id=$3`, role, time.Now(), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) UpdateRealname(ctx context.Context, id int64, status, realName, idHash, idMasked string, birth *time.Time, isMinor bool) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE auth_accounts SET
			realname_status=$1, real_name=$2, id_card_hash=$3, id_card_masked=$4,
			birth_date=$5, is_minor=$6, realname_verified_at=$7, updated_at=$7
		WHERE id=$8`,
		status, realName, nullIfEmpty(idHash), idMasked, birth, isMinor, now, id)
	return err
}

// UpsertRefresh 写入新 refresh；若账号已有记录，将旧 hash 保留为 prev，宽限至 graceUntil（仅 UPDATE 路径）。
func (s *Store) UpsertRefresh(ctx context.Context, accountID int64, tokenHash string, expiresAt, graceUntil time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO auth_refresh_tokens (
			account_id, token_hash, expires_at, prev_token_hash, prev_expires_at, created_at, updated_at
		) VALUES ($1,$2,$3,NULL,NULL,now(),now())
		ON CONFLICT (account_id) DO UPDATE SET
			prev_token_hash = auth_refresh_tokens.token_hash,
			prev_expires_at = $4,
			token_hash = EXCLUDED.token_hash,
			expires_at = EXCLUDED.expires_at,
			updated_at = now()`, accountID, tokenHash, expiresAt, graceUntil)
	return err
}

// GetRefreshByHash 匹配当前 token_hash，或宽限期内的 prev_token_hash（解决轮换后客户端未落盘）。
func (s *Store) GetRefreshByHash(ctx context.Context, tokenHash string) (*RefreshRecord, error) {
	var r RefreshRecord
	err := s.db.GetContext(ctx, &r, `
		SELECT account_id, token_hash, expires_at, prev_token_hash, prev_expires_at, created_at, updated_at
		FROM auth_refresh_tokens
		WHERE token_hash = $1
		   OR (prev_token_hash = $1 AND prev_expires_at IS NOT NULL AND prev_expires_at > now())
		ORDER BY CASE WHEN token_hash = $1 THEN 0 ELSE 1 END
		LIMIT 1`, tokenHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &r, err
}

func (s *Store) DeleteRefreshByAccount(ctx context.Context, accountID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM auth_refresh_tokens WHERE account_id=$1`, accountID)
	return err
}

func (s *Store) DeleteRefreshByHash(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM auth_refresh_tokens WHERE token_hash=$1`, tokenHash)
	return err
}

func (s *Store) FindOAuthByUnionID(ctx context.Context, provider, unionID string) (*OAuthIdentity, error) {
	if unionID == "" {
		return nil, ErrNotFound
	}
	var o OAuthIdentity
	err := s.db.GetContext(ctx, &o, `
		SELECT id, account_id, provider, platform, open_id, union_id, created_at, updated_at
		FROM auth_oauth_identities WHERE provider=$1 AND union_id=$2 LIMIT 1`, provider, unionID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &o, err
}

func (s *Store) FindOAuthByOpenID(ctx context.Context, provider, platform, openID string) (*OAuthIdentity, error) {
	var o OAuthIdentity
	err := s.db.GetContext(ctx, &o, `
		SELECT id, account_id, provider, platform, open_id, union_id, created_at, updated_at
		FROM auth_oauth_identities WHERE provider=$1 AND platform=$2 AND open_id=$3`, provider, platform, openID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &o, err
}

func (s *Store) CreateOAuthIdentity(ctx context.Context, accountID int64, provider, platform, openID, unionID string) error {
	var uid *string
	if unionID != "" {
		uid = &unionID
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO auth_oauth_identities (account_id, provider, platform, open_id, union_id)
		VALUES ($1,$2,$3,$4,$5)`, accountID, provider, platform, openID, uid)
	return err
}

func (s *Store) EnsureAdminSeed(ctx context.Context, phone, nickname string) error {
	if phone == "" {
		return nil
	}
	_, err := s.GetAccountByPhone(ctx, phone)
	if err == nil {
		return nil
	}
	if !errors.Is(err, ErrNotFound) {
		return err
	}
	p := phone
	_, err = s.CreateAccount(ctx, CreateAccountParams{
		PrincipalType: "admin",
		Phone:         &p,
		Role:          "platform_admin",
		Nickname:      nickname,
	})
	if err != nil {
		return fmt.Errorf("seed admin: %w", err)
	}
	return nil
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
