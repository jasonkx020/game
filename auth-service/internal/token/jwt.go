package token

import (
	"crypto/ecdsa"
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
	privateKey *ecdsa.PrivateKey
	publicKey  *ecdsa.PublicKey
	kid        string
	ttl        time.Duration
}

func NewManager(privateKey *ecdsa.PrivateKey, kid string, accessTTL time.Duration) *Manager {
	if kid == "" {
		kid = "es256-1"
	}
	return &Manager{
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
		kid:        kid,
		ttl:        accessTTL,
	}
}

// NewVerifier builds a parse-only manager (tests / callers with public key only).
func NewVerifier(publicKey *ecdsa.PublicKey, kid string, accessTTL time.Duration) *Manager {
	if kid == "" {
		kid = "es256-1"
	}
	return &Manager{publicKey: publicKey, kid: kid, ttl: accessTTL}
}

func (m *Manager) KeyID() string { return m.kid }

func (m *Manager) PublicKey() *ecdsa.PublicKey { return m.publicKey }

func (m *Manager) IssueAccess(accountID int64, role, principalType, phone, nickname string) (string, int, error) {
	if m.privateKey == nil {
		return "", 0, errors.New("private key required to issue tokens")
	}
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
	t := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	t.Header["kid"] = m.kid
	s, err := t.SignedString(m.privateKey)
	if err != nil {
		return "", 0, err
	}
	return s, int(m.ttl.Seconds()), nil
}

// Parse validates signature and claims including expiry.
func (m *Manager) Parse(tokenStr string) (*Claims, error) {
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodES256.Alg()}))
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
		jwt.WithValidMethods([]string{jwt.SigningMethodES256.Alg()}),
		jwt.WithoutClaimsValidation(),
	)
	t, err := parser.Parse(tokenStr, m.keyFunc)
	if err != nil {
		return nil, err
	}
	return mapClaims(t.Claims)
}

func (m *Manager) keyFunc(t *jwt.Token) (interface{}, error) {
	if t.Method != jwt.SigningMethodES256 {
		return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
	}
	if m.publicKey == nil {
		return nil, errors.New("public key not configured")
	}
	return m.publicKey, nil
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
