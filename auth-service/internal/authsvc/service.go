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
	ErrNotFound     = errors.New("not found")
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
	IsNewUser        bool   `json:"is_new_user"`
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
	refreshGrace    time.Duration // 轮换后旧 refresh 仍可换票的宽限，缓解客户端未落盘
	defaultNickname string
}

func New(
	st *store.Store,
	tokens *token.Manager,
	otpStore *otp.Store,
	wx *wechat.Resolver,
	verifier kyc.Verifier,
	refreshTTL time.Duration,
	refreshGrace time.Duration,
	defaultNickname string,
) *Service {
	if refreshGrace <= 0 {
		refreshGrace = 2 * time.Minute
	}
	return &Service{
		store:           st,
		tokens:          tokens,
		otp:             otpStore,
		wechat:          wx,
		kyc:             verifier,
		refreshTTL:      refreshTTL,
		refreshGrace:    refreshGrace,
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
	if purpose == "" {
		purpose = otp.PurposeSignIn
	}
	// signin/login/register：只发码+频控，不预检账号是否存在
	if _, err := otp.NormalizePurpose(purpose); err != nil {
		return fmt.Errorf("%w: invalid purpose", ErrBadRequest)
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

// SignIn 玩家统一入口：验证码通过后无号建号、有号登录。
func (s *Service) SignIn(ctx context.Context, channel, target, code, nickname string) (*TokenPair, error) {
	channel = strings.ToLower(strings.TrimSpace(channel))
	if channel == "" {
		channel = otp.ChannelPhone
	}
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
	if err := s.otp.Verify(ctx, channel, target, otp.PurposeSignIn, code); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnauthorized, err)
	}

	acc, err := s.getByChannel(ctx, channel, target)
	isNew := false
	if errors.Is(err, store.ErrNotFound) {
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
		acc, err = s.store.CreateAccount(ctx, params)
		if err != nil {
			return nil, err
		}
		isNew = true
	} else if err != nil {
		return nil, err
	}

	if acc.PrincipalType != token.PrincipalPlayer {
		return nil, fmt.Errorf("%w: principal mismatch", ErrForbidden)
	}
	if acc.Status != "active" {
		return nil, fmt.Errorf("%w: account disabled", ErrForbidden)
	}
	_ = s.store.TouchLogin(ctx, acc.ID)
	pair, err := s.issuePair(ctx, acc)
	if err != nil {
		return nil, err
	}
	pair.IsNewUser = isNew
	return pair, nil
}

// Register deprecated：转调 SignIn。
func (s *Service) Register(ctx context.Context, channel, target, code, nickname string) (*TokenPair, error) {
	return s.SignIn(ctx, channel, target, code, nickname)
}

// Login：玩家转调 SignIn；Admin 仅已有账号登录、不建号。
func (s *Service) Login(ctx context.Context, channel, target, code, principalType string) (*TokenPair, error) {
	if principalType == token.PrincipalPlayer {
		return s.SignIn(ctx, channel, target, code, "")
	}
	if principalType != token.PrincipalAdmin {
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
		return nil, fmt.Errorf("%w: admin email login unsupported", ErrBadRequest)
	} else {
		return nil, fmt.Errorf("%w: invalid channel", ErrBadRequest)
	}
	if target == "" {
		return nil, fmt.Errorf("%w: target required", ErrBadRequest)
	}

	if err := s.otp.Verify(ctx, channel, target, otp.PurposeSignIn, code); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnauthorized, err)
	}

	acc, err := s.getByChannel(ctx, channel, target)
	if errors.Is(err, store.ErrNotFound) {
		return nil, fmt.Errorf("%w: account not found", ErrUnauthorized)
	}
	if err != nil {
		return nil, err
	}
	if acc.PrincipalType != token.PrincipalAdmin {
		return nil, fmt.Errorf("%w: principal mismatch", ErrForbidden)
	}
	if acc.Status != "active" {
		return nil, fmt.Errorf("%w: account disabled", ErrForbidden)
	}
	_ = s.store.TouchLogin(ctx, acc.ID)
	pair, err := s.issuePair(ctx, acc)
	if err != nil {
		return nil, err
	}
	pair.IsNewUser = false
	return pair, nil
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
	isNew := false
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
		isNew = true
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
	pair, err := s.issuePair(ctx, acc)
	if err != nil {
		return nil, err
	}
	pair.IsNewUser = isNew
	return pair, nil
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

// SetAccountStatus updates auth_accounts.status (active|disabled) for ops moderation.
func (s *Service) SetAccountStatus(ctx context.Context, accountID int64, status string) (*store.Account, error) {
	if accountID <= 0 {
		return nil, fmt.Errorf("%w: invalid account_id", ErrBadRequest)
	}
	if status != "active" && status != "disabled" {
		return nil, fmt.Errorf("%w: status must be active or disabled", ErrBadRequest)
	}
	if err := s.store.UpdateStatus(ctx, accountID, status); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, fmt.Errorf("%w: account not found", ErrNotFound)
		}
		return nil, err
	}
	acc, err := s.store.GetAccountByID(ctx, accountID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, fmt.Errorf("%w: account not found", ErrNotFound)
		}
		return nil, err
	}
	return acc, nil
}

// RevokeSessions deletes all refresh tokens for an account (access JWT still expires naturally).
func (s *Service) RevokeSessions(ctx context.Context, accountID int64) error {
	if accountID <= 0 {
		return fmt.Errorf("%w: invalid account_id", ErrBadRequest)
	}
	if _, err := s.store.GetAccountByID(ctx, accountID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return fmt.Errorf("%w: account not found", ErrNotFound)
		}
		return err
	}
	return s.store.DeleteRefreshByAccount(ctx, accountID)
}

func validAdminRole(role string) bool {
	switch role {
	case token.RolePlatformAdmin, token.RoleClubAdmin, "agent":
		return true
	default:
		return false
	}
}

// CreateAdminAccount creates principal_type=admin account for ops onboarding.
func (s *Service) CreateAdminAccount(ctx context.Context, phone, nickname, role string) (*store.Account, error) {
	phone = normalizePhone(phone)
	nickname = strings.TrimSpace(nickname)
	role = strings.TrimSpace(role)
	if phone == "" || len(phone) < 8 {
		return nil, fmt.Errorf("%w: invalid phone", ErrBadRequest)
	}
	if !validAdminRole(role) {
		return nil, fmt.Errorf("%w: invalid role", ErrBadRequest)
	}
	if nickname == "" {
		nickname = "管理员"
	}
	if _, err := s.store.GetAccountByPhone(ctx, phone); err == nil {
		return nil, fmt.Errorf("%w: phone already registered", ErrConflict)
	} else if !errors.Is(err, store.ErrNotFound) {
		return nil, err
	}
	p := phone
	acc, err := s.store.CreateAccount(ctx, store.CreateAccountParams{
		PrincipalType: token.PrincipalAdmin,
		Phone:         &p,
		Role:          role,
		Nickname:      nickname,
	})
	if err != nil {
		if store.IsUniqueViolation(err) {
			return nil, fmt.Errorf("%w: phone already registered", ErrConflict)
		}
		return nil, err
	}
	return acc, nil
}

// SetAccountRole updates auth_accounts.role for admin principals.
func (s *Service) SetAccountRole(ctx context.Context, accountID int64, role string) (*store.Account, error) {
	if accountID <= 0 {
		return nil, fmt.Errorf("%w: invalid account_id", ErrBadRequest)
	}
	role = strings.TrimSpace(role)
	if !validAdminRole(role) {
		return nil, fmt.Errorf("%w: invalid role", ErrBadRequest)
	}
	acc, err := s.store.GetAccountByID(ctx, accountID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, fmt.Errorf("%w: account not found", ErrNotFound)
		}
		return nil, err
	}
	if acc.PrincipalType != token.PrincipalAdmin {
		return nil, fmt.Errorf("%w: not an admin account", ErrBadRequest)
	}
	if err := s.store.UpdateRole(ctx, accountID, role); err != nil {
		return nil, err
	}
	acc.Role = role
	return acc, nil
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
	now := time.Now()
	exp := now.Add(s.refreshTTL)
	graceUntil := now.Add(s.refreshGrace)

	var rawRefresh string
	for attempt := 0; attempt < 3; attempt++ {
		rawRefresh, err = refreshtoken.NewOpaque()
		if err != nil {
			return nil, err
		}
		err = s.store.UpsertRefresh(ctx, acc.ID, refreshtoken.Hash(rawRefresh), exp, graceUntil)
		if err == nil {
			break
		}
		// 极端：token_hash 唯一冲突时重生 opaque 再写
		if store.IsUniqueViolation(err) && attempt < 2 {
			continue
		}
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
