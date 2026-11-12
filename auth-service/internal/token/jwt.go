package token

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	PrincipalPlayer = "player"
	PrincipalAdmin  = "admin"

	RolePlayer        = "player"
	RoleClubAdmin     = "club_admin"
	RolePlatformAdmin = "platform_admin"
)

type Claims struct {
	AccountID     int64
	Role          string
	PrincipalType string
	Phone         string
	Nickname      string
	ExpiresAt     time.Time
	IssuedAt      time.Time
}

type Manager struct {
	secret []byte
	ttl    time.Duration
}

func NewManager(secret string, accessTTL time.Duration) *Manager {
	return &Manager{secret: []byte(secret), ttl: accessTTL}
}

func (m *Manager) IssueAccess(accountID int64, role, principalType, phone, nickname string) (string, int, error) {
	now := time.Now()
	exp := now.Add(m.ttl)
	claims := jwt.MapClaims{
		"sub":  accountID,
		"role": role,
		"typ":  principalType,
		"iat":  now.Unix(),
		"exp":  exp.Unix(),
	}
	if phone != "" {
		claims["phone"] = phone
	}
	if nickname != "" {
		claims["nickname"] = nickname
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := t.SignedString(m.secret)
	if err != nil {
		return "", 0, err
	}
	return s, int(m.ttl.Seconds()), nil
}

// Parse validates signature and claims including expiry.
func (m *Manager) Parse(tokenStr string) (*Claims, error) {
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	t, err := parser.Parse(tokenStr, m.keyFunc)
	if err != nil {
		return nil, err
	}
	if !t.Valid {
		return nil, errors.New("invalid token")
	}
	return mapClaims(t.Claims)
}

// ParseAllowExpired validates signature and structure but ignores exp.
func (m *Manager) ParseAllowExpired(tokenStr string) (*Claims, error) {
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithoutClaimsValidation(),
	)
	t, err := parser.Parse(tokenStr, m.keyFunc)
	if err != nil {
		return nil, err
	}
	return mapClaims(t.Claims)
}

func (m *Manager) keyFunc(t *jwt.Token) (interface{}, error) {
	if t.Method != jwt.SigningMethodHS256 {
		return nil, fmt.Errorf("unexpected signing method")
	}
	return m.secret, nil
}

func mapClaims(raw jwt.Claims) (*Claims, error) {
	mc, ok := raw.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}
	sub, err := claimInt64(mc["sub"])
	if err != nil {
		return nil, errors.New("invalid sub")
	}
	role, _ := mc["role"].(string)
	typ, _ := mc["typ"].(string)
	phone, _ := mc["phone"].(string)
	nickname, _ := mc["nickname"].(string)
	if typ == "" {
		typ = PrincipalPlayer
	}
	if role == "" {
		role = RolePlayer
	}
	var exp, iat time.Time
	if v, err := claimInt64(mc["exp"]); err == nil {
		exp = time.Unix(v, 0)
	}
	if v, err := claimInt64(mc["iat"]); err == nil {
		iat = time.Unix(v, 0)
	}
	return &Claims{
		AccountID:     sub,
		Role:          role,
		PrincipalType: typ,
		Phone:         phone,
		Nickname:      nickname,
		ExpiresAt:     exp,
		IssuedAt:      iat,
	}, nil
}

func claimInt64(v interface{}) (int64, error) {
	switch x := v.(type) {
	case float64:
		return int64(x), nil
	case int64:
		return x, nil
	case int:
		return int64(x), nil
	case string:
		return strconv.ParseInt(x, 10, 64)
	default:
		return 0, fmt.Errorf("unsupported type %T", v)
	}
}
