package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/example/auth-service/internal/token"
)

// requirePlatformAdmin parses Bearer access and requires typ=admin + role=platform_admin.
func (s *Server) requirePlatformAdmin(c *gin.Context) (*token.Claims, bool) {
	access := bearerToken(c)
	if access == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1001, "message": "token required", "request_id": c.GetString("request_id")})
		return nil, false
	}
	claims, _, err := s.auth.Introspect(c.Request.Context(), access)
	if err != nil {
		writeAuthErr(c, err)
		return nil, false
	}
	if claims.PrincipalType != token.PrincipalAdmin {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "admin required", "request_id": c.GetString("request_id")})
		return nil, false
	}
	if claims.Role != token.RolePlatformAdmin {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "platform_admin required", "request_id": c.GetString("request_id")})
		return nil, false
	}
	return claims, true
}

func (s *Server) createAdminAccount(c *gin.Context) {
	if _, ok := s.requirePlatformAdmin(c); !ok {
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
	acc, err := s.auth.CreateAdminAccount(c.Request.Context(), req.Phone, req.Nickname, req.Role)
	if err != nil {
		writeAuthErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"account_id":     acc.ID,
		"phone":          acc.PhoneStr(),
		"nickname":       acc.Nickname,
		"role":           acc.Role,
		"principal_type": acc.PrincipalType,
		"status":         acc.Status,
		"request_id":     c.GetString("request_id"),
	})
}

func (s *Server) patchAccountRole(c *gin.Context) {
	if _, ok := s.requirePlatformAdmin(c); !ok {
		return
	}
	accountID, err := strconv.ParseInt(c.Param("account_id"), 10, 64)
	if err != nil || accountID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid account_id", "request_id": c.GetString("request_id")})
		return
	}
	var req struct {
		Role string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	acc, err := s.auth.SetAccountRole(c.Request.Context(), accountID, strings.TrimSpace(req.Role))
	if err != nil {
		writeAuthErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"account_id": acc.ID,
		"role":       acc.Role,
		"request_id": c.GetString("request_id"),
	})
}

func (s *Server) patchAccountStatus(c *gin.Context) {
	if _, ok := s.requirePlatformAdmin(c); !ok {
		return
	}
	accountID, err := strconv.ParseInt(c.Param("account_id"), 10, 64)
	if err != nil || accountID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid account_id", "request_id": c.GetString("request_id")})
		return
	}
	var req struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	_ = req.Reason
	acc, err := s.auth.SetAccountStatus(c.Request.Context(), accountID, strings.TrimSpace(req.Status))
	if err != nil {
		writeAuthErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"account_id": acc.ID,
		"status":     acc.Status,
		"request_id": c.GetString("request_id"),
	})
}

func (s *Server) revokeAccountSessions(c *gin.Context) {
	if _, ok := s.requirePlatformAdmin(c); !ok {
		return
	}
	accountID, err := strconv.ParseInt(c.Param("account_id"), 10, 64)
	if err != nil || accountID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid account_id", "request_id": c.GetString("request_id")})
		return
	}
	if err := s.auth.RevokeSessions(c.Request.Context(), accountID); err != nil {
		writeAuthErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "account_id": accountID, "request_id": c.GetString("request_id")})
}
