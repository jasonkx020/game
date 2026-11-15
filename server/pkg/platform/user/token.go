package user

import (
	"errors"

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

func ParseUserID(tokenStr string, secret []byte) (int64, error) {
	claims, err := ParseAccessClaims(tokenStr, secret)
	if err != nil {
		return 0, err
	}
	return claims.AccountID, nil
}

func ParseAccessClaims(tokenStr string, secret []byte) (*AccessClaims, error) {
	raw, err := parseClaims(tokenStr, secret)
	if err != nil {
		return nil, err
	}
	sub, ok := raw["sub"].(float64)
	if !ok {
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
		AccountID:     int64(sub),
		Role:          role,
		PrincipalType: typ,
		Phone:         phone,
		Nickname:      nickname,
	}, nil
}

func ParseRole(tokenStr string, secret []byte) (string, error) {
	claims, err := parseClaims(tokenStr, secret)
	if err != nil {
		return "", err
	}
	role, _ := claims["role"].(string)
	if role == "" {
		return RolePlayer, nil
	}
	return role, nil
}

func ParsePrincipalType(tokenStr string, secret []byte) (string, error) {
	claims, err := parseClaims(tokenStr, secret)
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

func parseClaims(tokenStr string, secret []byte) (jwt.MapClaims, error) {
	t, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return secret, nil
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
