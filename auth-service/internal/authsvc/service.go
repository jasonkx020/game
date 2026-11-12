package authsvc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/example/auth-service/internal/kyc"
	"github.com/example/auth-service/internal/oauth/wechat"
	"github.com/example/auth-service/internal/otp"
	"github.com/example/auth-service/internal/refreshtoken"
	"github.com/example/auth-service/internal/store"
	"github.com/example/auth-service/internal/token"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrBadRequest   = errors.New("bad request")
	ErrForbidden    = errors.New("forbidden")
	ErrConflict     = errors.New("conflict")
)

type TokenPair struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	TokenType        string `json:"token_type"`
	AccountID        int64  `json:"account_id"`
	Role             string `json:"role"`
	Nickname         string `json:"nickname,omitempty"`
	Phone            string `json:"phone,omitempty"`
	Email            string `json:"email,omitempty"`
}

type Profile struct {
	AccountID      int64  `json:"account_id"`
	PrincipalType  string `json:"principal_type"`
	Role           string `json:"role"`
	Nickname       string `json:"nickname"`
	AvatarURL      string `json:"avatar_url"`
	Phone          string `json:"phone,omitempty"`
	Email          string `json:"email,omitempty"`
	Status         string `json:"status"`
	RealnameStatus string `json:"realname_status"`
	RealNameMasked string `json:"real_name_masked,omitempty"`
	IDCardMasked   string `json:"id_card_masked,omitempty"`
	IsMinor        bool   `json:"is_minor"`
}

type Service struct {
	store           *store.Store
	tokens          *token.Manager
	otp             *otp.Store
	wechat          *wechat.Resolver
	kyc             kyc.Verifier
	refreshTTL      time.Duration
	defaultNickname string
}

func New(
	st *store.Store,
	tokens *token.Manager,
	otpStore *otp.Store,
	wx *wechat.Resolver,
	verifier kyc.Verifier,
	refreshTTL time.Duration,
	defaultNickname string,
) *Service {
	return &Service{
		store:           st,
		tokens:          tokens,
		otp:             otpStore,
		wechat:          wx,
		kyc:             verifier,
		refreshTTL:      refreshTTL,
		defaultNickname: defaultNickname,
	}
}

func (s *Service) SendCode(ctx context.Context, channel, target, purpose string) error {
	channel = strings.ToLower(strings.TrimSpace(channel))
	purpose = strings.ToLower(strings.TrimSpace(purpose))
	target = strings.TrimSpace(target)
	if channel == otp.ChannelPhone {
		target = normalizePhone(target)
	} else if channel == otp.ChannelEmail {
		target = normalizeEmail(target)
	} else {
		return fmt.Errorf("%w: invalid channel", ErrBadRequest)
	}
	if target == "" {
		return fmt.Errorf("%w: target required", ErrBadRequest)
	}
	if purpose != otp.PurposeRegister && purpose != otp.PurposeLogin {
		return fmt.Errorf("%w: invalid purpose", ErrBadRequest)
	}

	if purpose == otp.PurposeRegister {
		if err := s.ensureTargetFree(ctx, channel, target); err != nil {
			return err
		}
	}
	if purpose == otp.PurposeLogin {
		if _, err := s.getByChannel(ctx, channel, target); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return fmt.Errorf("%w: account not found", ErrUnauthorized)
			}
			return err
		}
	}

	_, err := s.otp.Send(ctx, channel, target, purpose)
	if errors.Is(err, otp.ErrRateLimited) {
		return fmt.Errorf("%w: too many requests", ErrForbidden)
	}
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBadRequest, err)
	}
	return nil
}

func (s *Service) Register(ctx context.Context, channel, target, code, nickname string) (*TokenPair, error) {
	channel = strings.ToLower(strings.TrimSpace(channel))
	target = strings.TrimSpace(target)
	if channel == otp.ChannelPhone {
		target = normalizePhone(target)
	} else if channel == otp.ChannelEmail {
		target = normalizeEmail(target)
	} else {
		return nil, fmt.Errorf("%w: invalid channel", ErrBadRequest)
	}
	if target == "" {
		return nil, fmt.Errorf("%w: target required", ErrBadRequest)
	}
	if err := s.otp.Verify(ctx, channel, target, otp.PurposeRegister, code); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnauthorized, err)
	}
	if err := s.ensureTargetFree(ctx, channel, target); err != nil {
		return nil, err
	}

	nick := strings.TrimSpace(nickname)
	if nick == "" {
		if channel == otp.ChannelPhone && len(target) >= 4 {
			nick = "玩家" + target[len(target)-4:]
		} else {
			nick = "玩家"
		}
		if s.defaultNickname != "" {
			nick = s.defaultNickname
		}
	}

	params := store.CreateAccountParams{
		PrincipalType: token.PrincipalPlayer,
		Role:          token.RolePlayer,
		Nickname:      nick,
	}
	if channel == otp.ChannelPhone {
		params.Phone = &target
	} else {
		params.Email = &target
	}
	acc, err := s.store.CreateAccount(ctx, params)
	if err != nil {
		return nil, err
	}
	_ = s.store.TouchLogin(ctx, acc.ID)
	return s.issuePair(ctx, acc)
}

// Login authenticates an existing account (phone or email + code). Admin keeps phone-only.
func (s *Service) Login(ctx context.Context, channel, target, code, principalType string) (*TokenPair, error) {
	if principalType != token.PrincipalPlayer && principalType != token.PrincipalAdmin {
		return nil, fmt.Errorf("%w: invalid principal", ErrBadRequest)
	}
	channel = strings.ToLower(strings.TrimSpace(channel))
	if channel == "" {
		channel = otp.ChannelPhone
	}
	target = strings.TrimSpace(target)
	if channel == otp.ChannelPhone {
		target = normalizePhone(target)
	} else if channel == otp.ChannelEmail {
		if principalType == token.PrincipalAdmin {
			return nil, fmt.Errorf("%w: admin email login unsupported", ErrBadRequest)
		}
		target = normalizeEmail(target)
	} else {
		return nil, fmt.Errorf("%w: invalid channel", ErrBadRequest)
	}
	if target == "" {
		return nil, fmt.Errorf("%w: target required", ErrBadRequest)
	}

	purpose := otp.PurposeLogin
	if err := s.otp.Verify(ctx, channel, target, purpose, code); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnauthorized, err)
	}

	acc, err := s.getByChannel(ctx, channel, target)
	if errors.Is(err, store.ErrNotFound) {
		return nil, fmt.Errorf("%w: account not found", ErrUnauthorized)
	}
	if err != nil {
		return nil, err
	}
	if acc.PrincipalType != principalType {
		return nil, fmt.Errorf("%w: principal mismatch", ErrForbidden)
	}
	if acc.Status != "active" {
		return nil, fmt.Errorf("%w: account disabled", ErrForbidden)
	}
	_ = s.store.TouchLogin(ctx, acc.ID)
	return s.issuePair(ctx, acc)
}

func (s *Service) LoginWeChat(ctx context.Context, platform, code string) (*TokenPair, error) {
	id, err := s.wechat.Resolve(ctx, platform, code)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnauthorized, err)
	}

	var accountID int64
	if id.UnionID != "" {
		if o, err := s.store.FindOAuthByUnionID(ctx, wechat.ProviderWeChat, id.UnionID); err == nil {
			accountID = o.AccountID
		} else if !errors.Is(err, store.ErrNotFound) {
			return nil, err
		}
	}
	if accountID == 0 {
		if o, err := s.store.FindOAuthByOpenID(ctx, wechat.ProviderWeChat, id.Platform, id.OpenID); err == nil {
			accountID = o.AccountID
		} else if !errors.Is(err, store.ErrNotFound) {
			return nil, err
		}
	}

	var acc *store.Account
	if accountID == 0 {
		nick := id.Nickname
		if nick == "" {
			nick = "微信用户"
		}
		acc, err = s.store.CreateAccount(ctx, store.CreateAccountParams{
			PrincipalType: token.PrincipalPlayer,
			Role:          token.RolePlayer,
			Nickname:      nick,
			AvatarURL:     id.Avatar,
		})
		if err != nil {
			return nil, err
		}
		if err := s.store.CreateOAuthIdentity(ctx, acc.ID, wechat.ProviderWeChat, id.Platform, id.OpenID, id.UnionID); err != nil {
			return nil, err
		}
	} else {
		acc, err = s.store.GetAccountByID(ctx, accountID)
		if err != nil {
			return nil, err
		}
		// Ensure this platform openid is linked.
		if _, err := s.store.FindOAuthByOpenID(ctx, wechat.ProviderWeChat, id.Platform, id.OpenID); errors.Is(err, store.ErrNotFound) {
			_ = s.store.CreateOAuthIdentity(ctx, acc.ID, wechat.ProviderWeChat, id.Platform, id.OpenID, id.UnionID)
		}
	}
	if acc.Status != "active" {
		return nil, fmt.Errorf("%w: account disabled", ErrForbidden)
	}
	_ = s.store.TouchLogin(ctx, acc.ID)
	return s.issuePair(ctx, acc)
}

func (s *Service) VerifyRealname(ctx context.Context, accessToken, realName, idCardNo string) (*Profile, error) {
	claims, err := s.tokens.Parse(accessToken)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid access_token", ErrUnauthorized)
	}
	acc, err := s.store.GetAccountByID(ctx, claims.AccountID)
	if err != nil {
		return nil, fmt.Errorf("%w: account not found", ErrUnauthorized)
	}
	if acc.RealnameStatus == "verified" {
		return toProfile(acc), nil
	}

	res, err := s.kyc.Verify(ctx, realName, idCardNo)
	status := "failed"
	nameStore := ""
	idHash := ""
	idMasked := ""
	var birth *time.Time
	isMinor := false
	if res != nil {
		nameStore = res.NameMasked
		idHash = res.IDHash
		idMasked = res.IDMasked
		birth = res.BirthDate
		isMinor = res.IsMinor
		if res.Matched {
			status = "verified"
		}
	}
	_ = s.store.UpdateRealname(ctx, acc.ID, status, nameStore, idHash, idMasked, birth, isMinor)
	acc, _ = s.store.GetAccountByID(ctx, acc.ID)
	if errors.Is(err, kyc.ErrMismatch) || (res != nil && !res.Matched) {
		return toProfile(acc), fmt.Errorf("%w: identity mismatch", ErrForbidden)
	}
	if err != nil {
		return toProfile(acc), fmt.Errorf("%w: %v", ErrBadRequest, err)
	}
	return toProfile(acc), nil
}

func (s *Service) Me(ctx context.Context, accessToken string) (*Profile, error) {
	claims, err := s.tokens.Parse(accessToken)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid access_token", ErrUnauthorized)
	}
	acc, err := s.store.GetAccountByID(ctx, claims.AccountID)
	if err != nil {
		return nil, fmt.Errorf("%w: account not found", ErrUnauthorized)
	}
	return toProfile(acc), nil
}

func (s *Service) Refresh(ctx context.Context, accessToken, refreshToken, expectedPrincipal string) (*TokenPair, error) {
	if accessToken == "" || refreshToken == "" {
		return nil, fmt.Errorf("%w: access_token and refresh_token required", ErrBadRequest)
	}

	claims, err := s.tokens.ParseAllowExpired(accessToken)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid access_token", ErrUnauthorized)
	}
	if expectedPrincipal != "" && claims.PrincipalType != expectedPrincipal {
		return nil, fmt.Errorf("%w: principal type mismatch", ErrUnauthorized)
	}

	hash := refreshtoken.Hash(refreshToken)
	rec, err := s.store.GetRefreshByHash(ctx, hash)
	if errors.Is(err, store.ErrNotFound) {
		return nil, fmt.Errorf("%w: invalid refresh_token", ErrUnauthorized)
	}
	if err != nil {
		return nil, err
	}
	if time.Now().After(rec.ExpiresAt) {
		_ = s.store.DeleteRefreshByHash(ctx, hash)
		return nil, fmt.Errorf("%w: refresh_token expired", ErrUnauthorized)
	}
	if rec.AccountID != claims.AccountID {
		return nil, fmt.Errorf("%w: token subject mismatch", ErrUnauthorized)
	}

	acc, err := s.store.GetAccountByID(ctx, rec.AccountID)
	if errors.Is(err, store.ErrNotFound) {
		return nil, fmt.Errorf("%w: account not found", ErrUnauthorized)
	}
	if err != nil {
		return nil, err
	}
	if acc.Status != "active" {
		return nil, fmt.Errorf("%w: account disabled", ErrForbidden)
	}
	if expectedPrincipal != "" && acc.PrincipalType != expectedPrincipal {
		return nil, fmt.Errorf("%w: principal mismatch", ErrUnauthorized)
	}

	return s.issuePair(ctx, acc)
}

func (s *Service) Logout(ctx context.Context, accessToken string) error {
	if accessToken == "" {
		return fmt.Errorf("%w: missing token", ErrBadRequest)
	}
	claims, err := s.tokens.ParseAllowExpired(accessToken)
	if err != nil {
		return fmt.Errorf("%w: invalid access_token", ErrUnauthorized)
	}
	return s.store.DeleteRefreshByAccount(ctx, claims.AccountID)
}

func (s *Service) Introspect(ctx context.Context, accessToken string) (*token.Claims, *store.Account, error) {
	claims, err := s.tokens.Parse(accessToken)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: invalid access_token", ErrUnauthorized)
	}
	acc, err := s.store.GetAccountByID(ctx, claims.AccountID)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: account not found", ErrUnauthorized)
	}
	return claims, acc, nil
}

func (s *Service) issuePair(ctx context.Context, acc *store.Account) (*TokenPair, error) {
	access, expiresIn, err := s.tokens.IssueAccess(acc.ID, acc.Role, acc.PrincipalType, acc.PhoneStr(), acc.Nickname)
	if err != nil {
		return nil, err
	}
	rawRefresh, err := refreshtoken.NewOpaque()
	if err != nil {
		return nil, err
	}
	exp := time.Now().Add(s.refreshTTL)
	if err := s.store.UpsertRefresh(ctx, acc.ID, refreshtoken.Hash(rawRefresh), exp); err != nil {
		return nil, err
	}
	return &TokenPair{
		AccessToken:      access,
		RefreshToken:     rawRefresh,
		ExpiresIn:        expiresIn,
		RefreshExpiresIn: int(s.refreshTTL.Seconds()),
		TokenType:        "Bearer",
		AccountID:        acc.ID,
		Role:             acc.Role,
		Nickname:         acc.Nickname,
		Phone:            acc.PhoneStr(),
		Email:            acc.EmailStr(),
	}, nil
}

func (s *Service) ensureTargetFree(ctx context.Context, channel, target string) error {
	_, err := s.getByChannel(ctx, channel, target)
	if err == nil {
		return fmt.Errorf("%w: already registered", ErrConflict)
	}
	if !errors.Is(err, store.ErrNotFound) {
		return err
	}
	return nil
}

func (s *Service) getByChannel(ctx context.Context, channel, target string) (*store.Account, error) {
	if channel == otp.ChannelEmail {
		return s.store.GetAccountByEmail(ctx, target)
	}
	return s.store.GetAccountByPhone(ctx, target)
}

func toProfile(acc *store.Account) *Profile {
	if acc == nil {
		return nil
	}
	p := &Profile{
		AccountID:      acc.ID,
		PrincipalType:  acc.PrincipalType,
		Role:           acc.Role,
		Nickname:       acc.Nickname,
		AvatarURL:      acc.AvatarURL,
		Phone:          acc.PhoneStr(),
		Email:          acc.EmailStr(),
		Status:         acc.Status,
		RealnameStatus: acc.RealnameStatus,
		IDCardMasked:   acc.IDCardMasked,
		IsMinor:        acc.IsMinor,
	}
	if acc.RealnameStatus == "verified" && acc.RealName != "" {
		p.RealNameMasked = kyc.MaskName(acc.RealName)
	}
	return p
}

func normalizePhone(phone string) string {
	var b []rune
	for _, r := range phone {
		if unicode.IsDigit(r) {
			b = append(b, r)
		}
	}
	return string(b)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
