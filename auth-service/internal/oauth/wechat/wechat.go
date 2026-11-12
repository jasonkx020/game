package wechat

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	PlatformWeb         = "web"
	PlatformMiniProgram = "miniprogram"
	ProviderWeChat      = "wechat"
)

type Identity struct {
	OpenID   string
	UnionID  string
	Nickname string
	Avatar   string
	Platform string
}

type Config struct {
	Mode      string // mock | live
	MPAppID   string
	MPSecret  string
	WebAppID  string
	WebSecret string
	HTTP      *http.Client
}

type Resolver struct {
	cfg Config
}

func NewResolver(cfg Config) *Resolver {
	if cfg.HTTP == nil {
		cfg.HTTP = &http.Client{Timeout: 10 * time.Second}
	}
	if cfg.Mode == "" {
		if cfg.MPAppID == "" && cfg.WebAppID == "" {
			cfg.Mode = "mock"
		} else {
			cfg.Mode = "live"
		}
	}
	return &Resolver{cfg: cfg}
}

func (r *Resolver) Resolve(ctx context.Context, platform, code string) (*Identity, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, fmt.Errorf("code required")
	}
	if platform != PlatformWeb && platform != PlatformMiniProgram {
		return nil, fmt.Errorf("invalid platform")
	}
	if r.cfg.Mode == "mock" {
		return r.mock(platform, code), nil
	}
	switch platform {
	case PlatformMiniProgram:
		return r.resolveMP(ctx, code)
	default:
		return r.resolveWeb(ctx, code)
	}
}

func (r *Resolver) mock(platform, code string) *Identity {
	sum := sha256.Sum256([]byte(platform + ":" + code))
	hexID := hex.EncodeToString(sum[:])
	return &Identity{
		OpenID:   "mock_openid_" + hexID[:16],
		UnionID:  "mock_union_" + hexID[16:32],
		Nickname: "微信用户",
		Avatar:   "",
		Platform: platform,
	}
}

func (r *Resolver) resolveMP(ctx context.Context, code string) (*Identity, error) {
	if r.cfg.MPAppID == "" || r.cfg.MPSecret == "" {
		return nil, fmt.Errorf("wechat miniprogram credentials not configured")
	}
	u := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		url.QueryEscape(r.cfg.MPAppID), url.QueryEscape(r.cfg.MPSecret), url.QueryEscape(code),
	)
	var resp struct {
		OpenID     string `json:"openid"`
		UnionID    string `json:"unionid"`
		SessionKey string `json:"session_key"`
		ErrCode    int    `json:"errcode"`
		ErrMsg     string `json:"errmsg"`
	}
	if err := r.getJSON(ctx, u, &resp); err != nil {
		return nil, err
	}
	if resp.ErrCode != 0 || resp.OpenID == "" {
		return nil, fmt.Errorf("wechat mp: %d %s", resp.ErrCode, resp.ErrMsg)
	}
	return &Identity{OpenID: resp.OpenID, UnionID: resp.UnionID, Nickname: "微信用户", Platform: PlatformMiniProgram}, nil
}

func (r *Resolver) resolveWeb(ctx context.Context, code string) (*Identity, error) {
	if r.cfg.WebAppID == "" || r.cfg.WebSecret == "" {
		return nil, fmt.Errorf("wechat web credentials not configured")
	}
	tokenURL := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		url.QueryEscape(r.cfg.WebAppID), url.QueryEscape(r.cfg.WebSecret), url.QueryEscape(code),
	)
	var tok struct {
		AccessToken string `json:"access_token"`
		OpenID      string `json:"openid"`
		UnionID     string `json:"unionid"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := r.getJSON(ctx, tokenURL, &tok); err != nil {
		return nil, err
	}
	if tok.ErrCode != 0 || tok.AccessToken == "" || tok.OpenID == "" {
		return nil, fmt.Errorf("wechat web token: %d %s", tok.ErrCode, tok.ErrMsg)
	}
	infoURL := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s",
		url.QueryEscape(tok.AccessToken), url.QueryEscape(tok.OpenID),
	)
	var info struct {
		Nickname string `json:"nickname"`
		HeadImg  string `json:"headimgurl"`
		UnionID  string `json:"unionid"`
		ErrCode  int    `json:"errcode"`
		ErrMsg   string `json:"errmsg"`
	}
	_ = r.getJSON(ctx, infoURL, &info)
	union := tok.UnionID
	if info.UnionID != "" {
		union = info.UnionID
	}
	nick := info.Nickname
	if nick == "" {
		nick = "微信用户"
	}
	return &Identity{
		OpenID:   tok.OpenID,
		UnionID:  union,
		Nickname: nick,
		Avatar:   info.HeadImg,
		Platform: PlatformWeb,
	}, nil
}

func (r *Resolver) getJSON(ctx context.Context, rawURL string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	res, err := r.cfg.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}
