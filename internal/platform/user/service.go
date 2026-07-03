package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jmoiron/sqlx"
)

var ErrInvalidSMS = errors.New("invalid sms code")

const (
	RolePlayer         = "player"
	RoleClubAdmin      = "club_admin"
	RolePlatformAdmin  = "platform_admin"
	RoleAgent          = "agent"
)

type User struct {
	ID       int64  `db:"id"`
	Phone    string `db:"phone"`
	Nickname string `db:"nickname"`
	Role     string `db:"role"`
}

type Service struct {
	db        *sqlx.DB
	jwtSecret []byte
	devCode   string
}

func NewService(db *sqlx.DB, jwtSecret, devCode string) *Service {
	return &Service{db: db, jwtSecret: []byte(jwtSecret), devCode: devCode}
}

func (s *Service) Login(ctx context.Context, phone, smsCode string) (*User, string, error) {
	if smsCode != s.devCode {
		return nil, "", ErrInvalidSMS
	}
	var u User
	err := s.db.GetContext(ctx, &u, `SELECT id, phone, nickname, role FROM users WHERE phone=$1`, phone)
	if err != nil {
		nickname := fmt.Sprintf("玩家%s", phone[len(phone)-4:])
		err = s.db.QueryRowxContext(ctx,
			`INSERT INTO users (phone, nickname, role) VALUES ($1, $2, 'player') RETURNING id, phone, nickname, role`,
			phone, nickname).StructScan(&u)
		if err != nil {
			return nil, "", err
		}
		_, _ = s.db.ExecContext(ctx,
			`INSERT INTO wallet_room_card (user_id, balance) VALUES ($1, 10) ON CONFLICT DO NOTHING`, u.ID)
		_, _ = s.db.ExecContext(ctx,
			`INSERT INTO wallet_game_coin (user_id, game_id, balance) VALUES ($1, 'dawugui', 0) ON CONFLICT DO NOTHING`, u.ID)
	}
	token, err := s.issueToken(u.ID, u.Role)
	if err != nil {
		return nil, "", err
	}
	return &u, token, nil
}

func (s *Service) Refresh(ctx context.Context, userID int64) (string, error) {
	u, err := s.GetByID(ctx, userID)
	if err != nil {
		return "", err
	}
	return s.issueToken(u.ID, u.Role)
}

func (s *Service) GetByID(ctx context.Context, userID int64) (*User, error) {
	var u User
	if err := s.db.GetContext(ctx, &u, `SELECT id, phone, nickname, role FROM users WHERE id=$1`, userID); err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Service) issueToken(userID int64, role string) (string, error) {
	claims := jwt.MapClaims{
		"sub":  userID,
		"role": role,
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
		"iat":  time.Now().Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(s.jwtSecret)
}

func ParseUserID(tokenStr string, secret []byte) (int64, error) {
	t, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil || !t.Valid {
		return 0, err
	}
	claims, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return 0, errors.New("invalid claims")
	}
	sub, ok := claims["sub"].(float64)
	if !ok {
		return 0, errors.New("invalid sub")
	}
	return int64(sub), nil
}

func ParseRole(tokenStr string, secret []byte) (string, error) {
	t, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil || !t.Valid {
		return "", err
	}
	claims, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid claims")
	}
	role, _ := claims["role"].(string)
	if role == "" {
		return RolePlayer, nil
	}
	return role, nil
}
