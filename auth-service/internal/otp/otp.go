package otp

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

var (
	ErrInvalidCode = errors.New("invalid verification code")
	ErrRateLimited = errors.New("rate limited")
)

const (
	ChannelPhone = "phone"
	ChannelEmail = "email"

	PurposeRegister = "register"
	PurposeLogin    = "login"
)

type Store struct {
	rdb           *goredis.Client
	devSMSCode    string
	devEmailCode  string
	ttl           time.Duration
	rateLimitTTL  time.Duration
}

func New(rdb *goredis.Client, devSMS, devEmail string) *Store {
	return &Store{
		rdb:          rdb,
		devSMSCode:   devSMS,
		devEmailCode: devEmail,
		ttl:          10 * time.Minute,
		rateLimitTTL: 60 * time.Second,
	}
}

func (s *Store) Send(ctx context.Context, channel, target, purpose string) (string, error) {
	if channel != ChannelPhone && channel != ChannelEmail {
		return "", fmt.Errorf("invalid channel")
	}
	if purpose != PurposeRegister && purpose != PurposeLogin {
		return "", fmt.Errorf("invalid purpose")
	}
	if target == "" {
		return "", fmt.Errorf("target required")
	}

	if s.rdb != nil {
		rlKey := fmt.Sprintf("auth:code:rl:%s:%s:%s", purpose, channel, target)
		ok, err := s.rdb.SetNX(ctx, rlKey, 1, s.rateLimitTTL).Result()
		if err != nil {
			return "", err
		}
		if !ok {
			return "", ErrRateLimited
		}
	}

	code, err := s.generateCode(channel)
	if err != nil {
		return "", err
	}

	if s.rdb != nil {
		key := codeKey(purpose, channel, target)
		if err := s.rdb.Set(ctx, key, hashCode(code), s.ttl).Err(); err != nil {
			return "", err
		}
	}

	// Dev: log code; production would call SMS/email provider here.
	log.Printf("otp send channel=%s purpose=%s target=%s code=%s", channel, purpose, maskTarget(channel, target), code)
	return code, nil
}

func (s *Store) Verify(ctx context.Context, channel, target, purpose, code string) error {
	if code == "" {
		return ErrInvalidCode
	}
	// Dev bypass
	if channel == ChannelPhone && s.devSMSCode != "" && code == s.devSMSCode {
		return nil
	}
	if channel == ChannelEmail && s.devEmailCode != "" && code == s.devEmailCode {
		return nil
	}
	if s.rdb == nil {
		return ErrInvalidCode
	}
	key := codeKey(purpose, channel, target)
	stored, err := s.rdb.Get(ctx, key).Result()
	if errors.Is(err, goredis.Nil) {
		return ErrInvalidCode
	}
	if err != nil {
		return err
	}
	if stored != hashCode(code) {
		return ErrInvalidCode
	}
	_ = s.rdb.Del(ctx, key)
	return nil
}

func (s *Store) generateCode(channel string) (string, error) {
	if channel == ChannelPhone && s.devSMSCode != "" {
		return s.devSMSCode, nil
	}
	if channel == ChannelEmail && s.devEmailCode != "" {
		return s.devEmailCode, nil
	}
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	n := int(b[0])<<16 | int(b[1])<<8 | int(b[2])
	return fmt.Sprintf("%06d", n%1000000), nil
}

func codeKey(purpose, channel, target string) string {
	return fmt.Sprintf("auth:code:%s:%s:%s", purpose, channel, target)
}

func hashCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

func maskTarget(channel, target string) string {
	if channel == ChannelPhone && len(target) >= 7 {
		return target[:3] + "****" + target[len(target)-4:]
	}
	if channel == ChannelEmail && len(target) > 4 {
		return target[:2] + "***"
	}
	return "***"
}
