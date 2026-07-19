package player

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jmoiron/sqlx"

	"github.com/example/game/internal/platform/user"
)

type Player struct {
	ID          int64      `db:"id"`
	Phone       string     `db:"phone"`
	Nickname    string     `db:"nickname"`
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

func (s *Service) Login(ctx context.Context, phone, smsCode string) (*Player, string, error) {
	if smsCode != s.devCode {
		return nil, "", user.ErrInvalidSMS
	}
	var adminExists bool
	if err := s.db.GetContext(ctx, &adminExists,
		`SELECT EXISTS(SELECT 1 FROM admin_users WHERE phone=$1)`, phone); err != nil {
		return nil, "", err
	}
	if adminExists {
		return nil, "", user.ErrInvalidSMS
	}
	var p Player
	err := s.db.GetContext(ctx, &p,
		`SELECT id, phone, nickname, avatar_url, status, last_login_at FROM players WHERE phone=$1`, phone)
	if err != nil {
		nickname := fmt.Sprintf("玩家%s", phone[len(phone)-4:])
		err = s.db.QueryRowxContext(ctx,
			`INSERT INTO players (phone, nickname) VALUES ($1, $2)
			 RETURNING id, phone, nickname, avatar_url, status, last_login_at`,
			phone, nickname).StructScan(&p)
		if err != nil {
			return nil, "", err
		}
		_, _ = s.db.ExecContext(ctx,
			`INSERT INTO wallet_room_card (user_id, balance) VALUES ($1, 10) ON CONFLICT DO NOTHING`, p.ID)
	}
	// 补齐游戏金币账户（含后上线的 liuzichong）
	_, _ = s.db.ExecContext(ctx,
		`INSERT INTO wallet_game_coin (user_id, game_id, balance) VALUES
		 ($1, 'dawugui', 0), ($1, 'liuzichong', 0) ON CONFLICT DO NOTHING`, p.ID)
	_, _ = s.db.ExecContext(ctx,
		`UPDATE players SET last_login_at=$1, updated_at=$1 WHERE id=$2`, time.Now(), p.ID)

	token, err := s.issueToken(p.ID)
	if err != nil {
		return nil, "", err
	}
	return &p, token, nil
}

func (s *Service) Refresh(ctx context.Context, playerID int64) (string, error) {
	if _, err := s.GetByID(ctx, playerID); err != nil {
		return "", err
	}
	return s.issueToken(playerID)
}

func (s *Service) GetByID(ctx context.Context, playerID int64) (*Player, error) {
	var p Player
	if err := s.db.GetContext(ctx, &p,
		`SELECT id, phone, nickname, avatar_url, status, last_login_at FROM players WHERE id=$1`, playerID); err != nil {
		return nil, err
	}
	return &p, nil
}

type UpdateProfileParams struct {
	Nickname  *string
	AvatarURL *string
}

func (s *Service) UpdateProfile(ctx context.Context, playerID int64, params UpdateProfileParams) (*Player, error) {
	p, err := s.GetByID(ctx, playerID)
	if err != nil {
		return nil, err
	}
	nickname := p.Nickname
	avatarURL := p.AvatarURL
	if params.Nickname != nil {
		nickname = *params.Nickname
	}
	if params.AvatarURL != nil {
		avatarURL = *params.AvatarURL
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE players SET nickname=$1, avatar_url=$2, updated_at=$3 WHERE id=$4`,
		nickname, avatarURL, time.Now(), playerID)
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, playerID)
}

func (s *Service) GetSettings(ctx context.Context, playerID int64) (map[string]interface{}, error) {
	var raw []byte
	err := s.db.GetContext(ctx, &raw, `SELECT settings FROM players WHERE id=$1`, playerID)
	if err != nil {
		return nil, err
	}
	out := map[string]interface{}{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	return out, nil
}

func (s *Service) UpdateSettings(ctx context.Context, playerID int64, patch map[string]interface{}) (map[string]interface{}, error) {
	cur, err := s.GetSettings(ctx, playerID)
	if err != nil {
		return nil, err
	}
	for k, v := range patch {
		cur[k] = v
	}
	b, _ := json.Marshal(cur)
	_, err = s.db.ExecContext(ctx,
		`UPDATE players SET settings=$1, updated_at=$2 WHERE id=$3`, b, time.Now(), playerID)
	if err != nil {
		return nil, err
	}
	return cur, nil
}

type PlayerRow struct {
	PlayerID    int64      `json:"player_id"`
	PhoneMasked string     `json:"phone_masked"`
	Nickname    string     `json:"nickname"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}

type playerDBRow struct {
	ID          int64      `db:"id"`
	Phone       string     `db:"phone"`
	Nickname    string     `db:"nickname"`
	Status      string     `db:"status"`
	CreatedAt   time.Time  `db:"created_at"`
	LastLoginAt *time.Time `db:"last_login_at"`
}

func (s *Service) ListAdmin(ctx context.Context, page, pageSize int) ([]PlayerRow, int, error) {
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
	if err := s.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM players`); err != nil {
		return nil, 0, err
	}

	var rows []playerDBRow
	err := s.db.SelectContext(ctx, &rows,
		`SELECT id, phone, nickname, status, created_at, last_login_at
		 FROM players ORDER BY id DESC LIMIT $1 OFFSET $2`,
		pageSize, offset)
	if err != nil {
		return nil, 0, err
	}

	out := make([]PlayerRow, len(rows))
	for i, r := range rows {
		out[i] = PlayerRow{
			PlayerID:    r.ID,
			PhoneMasked: user.MaskPhone(r.Phone),
			Nickname:    r.Nickname,
			Status:      r.Status,
			CreatedAt:   r.CreatedAt,
			LastLoginAt: r.LastLoginAt,
		}
	}
	return out, total, nil
}

func (s *Service) issueToken(playerID int64) (string, error) {
	claims := jwt.MapClaims{
		"sub":  playerID,
		"role": user.RolePlayer,
		"typ":  user.PrincipalTypePlayer,
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
		"iat":  time.Now().Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(s.jwtSecret)
}
