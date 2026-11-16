package user

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"strconv"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidSMS = errors.New("invalid sms code")

const (
	RolePlayer        = "player"
	RoleClubAdmin     = "club_admin"
	RolePlatformAdmin = "platform_admin"
	RoleAgent         = "agent"

	PrincipalTypeAdmin  = "admin"
	PrincipalTypePlayer = "player"
)

type AccessClaims struct {
	AccountID     int64
	Role          string
	PrincipalType string
	Phone         string
	Nickname      string
}

func ParseUserID(tokenStr string, pub *ecdsa.PublicKey) (int64, error) {
	claims, err := ParseAccessClaims(tokenStr, pub)
	if err != nil {
		return 0, err
	}
	return claims.AccountID, nil
}

func ParseAccessClaims(tokenStr string, pub *ecdsa.PublicKey) (*AccessClaims, error) {
	raw, err := parseClaims(tokenStr, pub)
	if err != nil {
		return nil, err
	}
	sub, err := claimInt64(raw["sub"])
	if err != nil {
		return nil, errors.New("invalid sub")
	}
	role, _ := raw["role"].(string)
	typ, _ := raw["typ"].(string)
	phone, _ := raw["phone"].(string)
	nickname, _ := raw["nickname"].(string)
	if typ == "" {
		typ = PrincipalTypePlayer
	}
	if role == "" {
		role = RolePlayer
	}
	return &AccessClaims{
		AccountID:     sub,
		Role:          role,
		PrincipalType: typ,
		Phone:         phone,
		Nickname:      nickname,
	}, nil
}

func ParseRole(tokenStr string, pub *ecdsa.PublicKey) (string, error) {
	claims, err := parseClaims(tokenStr, pub)
	if err != nil {
		return "", err
	}
	role, _ := claims["role"].(string)
	if role == "" {
		return RolePlayer, nil
	}
	return role, nil
}

func ParsePrincipalType(tokenStr string, pub *ecdsa.PublicKey) (string, error) {
	claims, err := parseClaims(tokenStr, pub)
	if err != nil {
		return "", err
	}
	typ, _ := claims["typ"].(string)
	if typ == "" {
		return PrincipalTypePlayer, nil
	}
	return typ, nil
}

func MaskPhone(phone string) string {
	if len(phone) < 7 {
		return phone
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}

func parseClaims(tokenStr string, pub *ecdsa.PublicKey) (jwt.MapClaims, error) {
	if pub == nil {
		return nil, errors.New("jwt public key required")
	}
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodES256.Alg()}))
	t, err := parser.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodES256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return pub, nil
	})
	if err != nil || !t.Valid {
		return nil, err
	}
	claims, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}
	return claims, nil
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
