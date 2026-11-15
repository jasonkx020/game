package api

import (
	"net/http"
	"strconv"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"

	"github.com/example/game/pkg/platform/moderation"
	"github.com/example/game/pkg/platform/user"
)

func (s *Server) adminListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	q := c.Query("q")
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	users, total, err := s.admins.ListAdmin(c.Request.Context(), page, pageSize, q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"users":     users,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func normalizeAdminPhone(phone string) string {
	var b []rune
	for _, r := range phone {
		if unicode.IsDigit(r) {
			b = append(b, r)
		}
	}
	return string(b)
}

func validAdminPhone(phone string) bool {
	if len(phone) < 8 || len(phone) > 15 {
		return false
	}
	for _, r := range phone {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func (s *Server) createAdminUser(c *gin.Context) {
	actorID, err := s.adminID(c)
	if err != nil {
		return
	}
	var req struct {
		Phone    string `json:"phone"`
		Nickname string `json:"nickname"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	phone := normalizeAdminPhone(req.Phone)
	nickname := strings.TrimSpace(req.Nickname)
	role := strings.TrimSpace(req.Role)
	if !validAdminPhone(phone) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid phone", "request_id": c.GetString("request_id")})
		return
	}
	switch role {
	case user.RolePlatformAdmin, user.RoleClubAdmin, user.RoleAgent:
	default:
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid role", "request_id": c.GetString("request_id")})
		return
	}
	if nickname == "" {
		nickname = "管理员"
	}
	if s.authClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": 503, "message": "auth client unavailable", "request_id": c.GetString("request_id")})
		return
	}
	authAcc, err := s.authClient.CreateAdminAccount(c.Request.Context(), bearerFromCtx(c), phone, nickname, role)
	if err != nil {
		msg := err.Error()
		code := http.StatusBadGateway
		if strings.Contains(msg, " 409 ") || strings.Contains(msg, "already registered") {
			code = http.StatusConflict
		}
		c.JSON(code, gin.H{"code": code, "message": "auth create: " + msg, "request_id": c.GetString("request_id")})
		return
	}
	u, err := s.admins.Create(c.Request.Context(), phone, nickname, role, authAcc.AccountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	_ = s.moderation.Record(c.Request.Context(), actorID, moderation.TargetAdmin, u.ID, moderation.ActionCreate, "", map[string]interface{}{
		"phone_masked":    user.MaskPhone(phone),
		"role":            role,
		"auth_account_id": authAcc.AccountID,
	})
	c.JSON(http.StatusOK, gin.H{
		"user_id":         u.ID,
		"auth_account_id": authAcc.AccountID,
		"phone_masked":    user.MaskPhone(u.Phone),
		"nickname":        u.Nickname,
		"role":            u.Role,
		"status":          u.Status,
		"request_id":      c.GetString("request_id"),
	})
}

func (s *Server) getAdminUser(c *gin.Context) {
	if _, err := s.adminID(c); err != nil {
		return
	}
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid user_id", "request_id": c.GetString("request_id")})
		return
	}
	u, err := s.admins.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "admin not found", "request_id": c.GetString("request_id")})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("audit_limit", "20"))
	actions, err := s.moderation.ListByTarget(c.Request.Context(), moderation.TargetAdmin, userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"user_id":         u.ID,
			"auth_account_id": u.AuthAccountID,
			"phone_masked":    user.MaskPhone(u.Phone),
			"nickname":        u.Nickname,
			"role":            u.Role,
			"status":          u.Status,
			"last_login_at":   u.LastLoginAt,
		},
		"actions":    actions,
		"request_id": c.GetString("request_id"),
	})
}

func (s *Server) patchAdminUser(c *gin.Context) {
	actorID, err := s.adminID(c)
	if err != nil {
		return
	}
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid user_id", "request_id": c.GetString("request_id")})
		return
	}
	var req struct {
		Nickname string `json:"nickname"`
		Reason   string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	nickname := strings.TrimSpace(req.Nickname)
	if nickname == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "nickname required", "request_id": c.GetString("request_id")})
		return
	}
	if err := s.admins.UpdateNickname(c.Request.Context(), userID, nickname); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	_ = s.moderation.Record(c.Request.Context(), actorID, moderation.TargetAdmin, userID, moderation.ActionUpdateNickname, req.Reason, map[string]interface{}{
		"nickname": nickname,
	})
	c.JSON(http.StatusOK, gin.H{"user_id": userID, "nickname": nickname, "request_id": c.GetString("request_id")})
}

func (s *Server) forceLogoutAdminUser(c *gin.Context) {
	actorID, err := s.adminID(c)
	if err != nil {
		return
	}
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid user_id", "request_id": c.GetString("request_id")})
		return
	}
	if userID == actorID {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "cannot force logout yourself", "request_id": c.GetString("request_id")})
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	u, err := s.admins.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "admin not found", "request_id": c.GetString("request_id")})
		return
	}
	if u.AuthAccountID == nil || s.authClient == nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "admin has no auth account", "request_id": c.GetString("request_id")})
		return
	}
	if err := s.authClient.RevokeSessions(c.Request.Context(), bearerFromCtx(c), *u.AuthAccountID); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": 502, "message": "auth revoke: " + err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	_ = s.moderation.Record(c.Request.Context(), actorID, moderation.TargetAdmin, userID, moderation.ActionForceLogout, req.Reason, nil)
	c.JSON(http.StatusOK, gin.H{"user_id": userID, "logged_out": true, "request_id": c.GetString("request_id")})
}

func (s *Server) adminListPlayers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	q := c.Query("q")
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	players, total, err := s.players.ListAdmin(c.Request.Context(), page, pageSize, q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"players":   players,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
