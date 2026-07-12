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

func ParseUserID(tokenStr string, secret []byte) (int64, error) {
	claims, err := parseClaims(tokenStr, secret)
	if err != nil {
		return 0, err
	}
	sub, ok := claims["sub"].(float64)
	if !ok {
		return 0, errors.New("invalid sub")
	}
	return int64(sub), nil
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
