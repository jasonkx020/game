package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"

	"github.com/example/auth-service/internal/authsvc"
	"github.com/example/auth-service/internal/config"
	"github.com/example/auth-service/internal/httpmw"
	"github.com/example/auth-service/internal/otp"
	"github.com/example/auth-service/internal/token"
)

type Server struct {
	cfg    *config.Config
	auth   *authsvc.Service
	rdb    *goredis.Client
	engine *gin.Engine
}

func New(cfg *config.Config, auth *authsvc.Service, rdb *goredis.Client) *Server {
	s := &Server{cfg: cfg, auth: auth, rdb: rdb}
	s.engine = s.build()
	return s
}

func (s *Server) Engine() *gin.Engine { return s.engine }

func (s *Server) build() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), httpmw.RequestID(), httpmw.CORS(s.cfg.CORSOrigins))

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := r.Group("/v1")
	v1.Use(httpmw.Signature(httpmw.SignatureConfig{
		AppID: s.cfg.AppID, AppSecret: s.cfg.HMACSecret, Skip: false,
	}, s.rdb))

	v1.POST("/auth/send-code", s.sendCode)
	v1.POST("/auth/sign-in", s.signIn)
	v1.POST("/auth/register", s.register) // deprecated → SignIn
	v1.POST("/auth/login", s.playerLogin) // deprecated alias → SignIn
	v1.POST("/auth/oauth/wechat", s.wechatLogin)
	v1.POST("/auth/refresh", s.playerRefresh)
	v1.POST("/auth/logout", s.logout)
	v1.GET("/auth/introspect", s.introspect)
	v1.GET("/auth/me", s.me)
	v1.POST("/auth/realname/verify", s.realnameVerify)

	v1.POST("/admin/auth/login", s.adminLogin)
	v1.POST("/admin/auth/refresh", s.adminRefresh)
	v1.POST("/admin/auth/logout", s.logout)

	// Ops moderation (platform_admin JWT + HMAC)
	v1.POST("/admin/auth/accounts", s.createAdminAccount)
	v1.PATCH("/admin/auth/accounts/:account_id/status", s.patchAccountStatus)
	v1.PATCH("/admin/auth/accounts/:account_id/role", s.patchAccountRole)
	v1.POST("/admin/auth/accounts/:account_id/revoke-sessions", s.revokeAccountSessions)

	return r
}

func (s *Server) sendCode(c *gin.Context) {
	var req struct {
		Channel string `json:"channel"`
		Target  string `json:"target"`
		Purpose string `json:"purpose"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	if err := s.auth.SendCode(c.Request.Context(), req.Channel, req.Target, req.Purpose); err != nil {
		writeAuthErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "request_id": c.GetString("request_id")})
}

func (s *Server) signIn(c *gin.Context) {
	channel, target, code, nickname, ok := parseSignInBody(c)
	if !ok {
		return
	}
	pair, err := s.auth.SignIn(c.Request.Context(), channel, target, code, nickname)
	if err != nil {
		writeAuthErr(c, err)
		return
	}
	writePair(c, pair)
}

func (s *Server) register(c *gin.Context) {
	s.signIn(c)
}

func (s *Server) playerLogin(c *gin.Context) {
	s.signIn(c)
}

func (s *Server) adminLogin(c *gin.Context) {
	var req struct {
		Channel string `json:"channel"`
		Target  string `json:"target"`
		Phone   string `json:"phone"`
		Code    string `json:"code"`
		SMSCode string `json:"sms_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	channel := req.Channel
	target := req.Target
	if target == "" && req.Phone != "" {
		channel = otp.ChannelPhone
		target = req.Phone
	}
	if channel == "" {
		channel = otp.ChannelPhone
	}
	code := req.Code
	if code == "" {
		code = req.SMSCode
	}
	pair, err := s.auth.Login(c.Request.Context(), channel, target, code, token.PrincipalAdmin)
	if err != nil {
		writeAuthErr(c, err)
		return
	}
	writePair(c, pair)
}

func parseSignInBody(c *gin.Context) (channel, target, code, nickname string, ok bool) {
	var req struct {
		Channel  string `json:"channel"`
		Target   string `json:"target"`
		Phone    string `json:"phone"`
		Email    string `json:"email"`
		Code     string `json:"code"`
		SMSCode  string `json:"sms_code"`
		Nickname string `json:"nickname"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
		return "", "", "", "", false
	}
	channel = req.Channel
	target = req.Target
	if target == "" {
		if req.Phone != "" {
			channel = otp.ChannelPhone
			target = req.Phone
		} else if req.Email != "" {
			channel = otp.ChannelEmail
			target = req.Email
		}
	}
	if channel == "" {
		channel = otp.ChannelPhone
	}
	code = req.Code
	if code == "" {
		code = req.SMSCode
	}
	return channel, target, code, req.Nickname, true
}

func (s *Server) wechatLogin(c *gin.Context) {
	var req struct {
		Platform string `json:"platform"`
		Code     string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	pair, err := s.auth.LoginWeChat(c.Request.Context(), req.Platform, req.Code)
	if err != nil {
		writeAuthErr(c, err)
		return
	}
	writePair(c, pair)
}

func (s *Server) playerRefresh(c *gin.Context) {
	s.refresh(c, token.PrincipalPlayer)
}

func (s *Server) adminRefresh(c *gin.Context) {
	s.refresh(c, token.PrincipalAdmin)
}

func (s *Server) refresh(c *gin.Context, principal string) {
	var req struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	pair, err := s.auth.Refresh(c.Request.Context(), req.AccessToken, req.RefreshToken, principal)
	if err != nil {
		writeAuthErr(c, err)
		return
	}
	writePair(c, pair)
}

func (s *Server) logout(c *gin.Context) {
	access := bearerOrBody(c)
	if err := s.auth.Logout(c.Request.Context(), access); err != nil {
		writeAuthErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "request_id": c.GetString("request_id")})
}

func (s *Server) introspect(c *gin.Context) {
	access := bearerToken(c)
	if access == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1001, "message": "token required", "request_id": c.GetString("request_id")})
		return
	}
	claims, acc, err := s.auth.Introspect(c.Request.Context(), access)
	if err != nil {
		writeAuthErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"account_id":      acc.ID,
		"role":            acc.Role,
		"principal_type":  claims.PrincipalType,
		"nickname":        acc.Nickname,
		"phone":           acc.PhoneStr(),
		"email":           acc.EmailStr(),
		"status":          acc.Status,
		"realname_status": acc.RealnameStatus,
		"exp":             claims.ExpiresAt.Unix(),
		"request_id":      c.GetString("request_id"),
	})
}

func (s *Server) me(c *gin.Context) {
	access := bearerToken(c)
	if access == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1001, "message": "token required", "request_id": c.GetString("request_id")})
		return
	}
	p, err := s.auth.Me(c.Request.Context(), access)
	if err != nil {
		writeAuthErr(c, err)
		return
	}
	writeProfile(c, p)
}

func (s *Server) realnameVerify(c *gin.Context) {
	access := bearerToken(c)
	if access == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1001, "message": "token required", "request_id": c.GetString("request_id")})
		return
	}
	var req struct {
		RealName string `json:"real_name"`
		IDCardNo string `json:"id_card_no"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	p, err := s.auth.VerifyRealname(c.Request.Context(), access, req.RealName, req.IDCardNo)
	if err != nil {
		if p != nil && errors.Is(err, authsvc.ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{
				"code":             403,
				"message":          err.Error(),
				"realname_status":  p.RealnameStatus,
				"real_name_masked": p.RealNameMasked,
				"id_card_masked":   p.IDCardMasked,
				"is_minor":         p.IsMinor,
				"request_id":       c.GetString("request_id"),
			})
			return
		}
		writeAuthErr(c, err)
		return
	}
	writeProfile(c, p)
}

func bearerToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	}
	return ""
}

func bearerOrBody(c *gin.Context) string {
	if t := bearerToken(c); t != "" {
		return t
	}
	var body struct {
		AccessToken string `json:"access_token"`
	}
	_ = c.ShouldBindJSON(&body)
	return body.AccessToken
}

func writePair(c *gin.Context, pair *authsvc.TokenPair) {
	c.JSON(http.StatusOK, gin.H{
		"access_token":       pair.AccessToken,
		"refresh_token":      pair.RefreshToken,
		"expires_in":         pair.ExpiresIn,
		"refresh_expires_in": pair.RefreshExpiresIn,
		"token_type":         pair.TokenType,
		"account_id":         pair.AccountID,
		"user_id":            pair.AccountID,
		"role":               pair.Role,
		"nickname":           pair.Nickname,
		"phone":              pair.Phone,
		"email":              pair.Email,
		"is_new_user":        pair.IsNewUser,
		"request_id":         c.GetString("request_id"),
	})
}

func writeProfile(c *gin.Context, p *authsvc.Profile) {
	status := http.StatusOK
	c.JSON(status, gin.H{
		"account_id":        p.AccountID,
		"principal_type":    p.PrincipalType,
		"role":              p.Role,
		"nickname":          p.Nickname,
		"avatar_url":        p.AvatarURL,
		"phone":             p.Phone,
		"email":             p.Email,
		"status":            p.Status,
		"realname_status":   p.RealnameStatus,
		"real_name_masked":  p.RealNameMasked,
		"id_card_masked":    p.IDCardMasked,
		"is_minor":          p.IsMinor,
		"request_id":        c.GetString("request_id"),
	})
}

func writeAuthErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, authsvc.ErrBadRequest):
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
	case errors.Is(err, authsvc.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": err.Error(), "request_id": c.GetString("request_id")})
	case errors.Is(err, authsvc.ErrConflict):
		c.JSON(http.StatusConflict, gin.H{"code": 409, "message": err.Error(), "request_id": c.GetString("request_id")})
	case errors.Is(err, authsvc.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": err.Error(), "request_id": c.GetString("request_id")})
	default:
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1001, "message": err.Error(), "request_id": c.GetString("request_id")})
	}
}
