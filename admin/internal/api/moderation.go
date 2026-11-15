package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/example/game/pkg/opskick"
	"github.com/example/game/pkg/platform/moderation"
	"github.com/example/game/pkg/platform/user"
)

func bearerFromCtx(c *gin.Context) string {
	token := c.GetString("access_token")
	if token != "" {
		return token
	}
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	}
	return ""
}

func (s *Server) banPlayer(c *gin.Context) {
	s.setPlayerBan(c, true)
}

func (s *Server) unbanPlayer(c *gin.Context) {
	s.setPlayerBan(c, false)
}

func (s *Server) setPlayerBan(c *gin.Context, ban bool) {
	actorID, err := s.adminID(c)
	if err != nil {
		return
	}
	playerID, err := strconv.ParseInt(c.Param("player_id"), 10, 64)
	if err != nil || playerID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid player_id", "request_id": c.GetString("request_id")})
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)

	p, err := s.players.GetByID(c.Request.Context(), playerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "player not found", "request_id": c.GetString("request_id")})
		return
	}

	status := "disabled"
	action := moderation.ActionBan
	if !ban {
		status = "active"
		action = moderation.ActionUnban
	}

	if p.AuthAccountID != nil && s.authClient != nil {
		token := bearerFromCtx(c)
		if err := s.authClient.SetAccountStatus(c.Request.Context(), token, *p.AuthAccountID, status, req.Reason); err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"code": 502, "message": "auth status: " + err.Error(), "request_id": c.GetString("request_id")})
			return
		}
		if ban {
			_ = s.authClient.RevokeSessions(c.Request.Context(), token, *p.AuthAccountID)
		}
	}

	if err := s.players.SetStatus(c.Request.Context(), playerID, status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}

	auditSN := s.audit.Next()
	_ = s.moderation.Record(c.Request.Context(), actorID, moderation.TargetPlayer, playerID, action, req.Reason, map[string]interface{}{
		"audit_sn": auditSN, "auth_account_id": p.AuthAccountID,
	})

	if ban {
		_ = opskick.Publish(c.Request.Context(), s.rdb, playerID, req.Reason, auditSN)
	}

	c.JSON(http.StatusOK, gin.H{
		"player_id": playerID, "status": status, "audit_sn": auditSN, "request_id": c.GetString("request_id"),
	})
}

func (s *Server) kickPlayer(c *gin.Context) {
	actorID, err := s.adminID(c)
	if err != nil {
		return
	}
	playerID, err := strconv.ParseInt(c.Param("player_id"), 10, 64)
	if err != nil || playerID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid player_id", "request_id": c.GetString("request_id")})
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)

	p, err := s.players.GetByID(c.Request.Context(), playerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "player not found", "request_id": c.GetString("request_id")})
		return
	}

	if p.AuthAccountID != nil && s.authClient != nil {
		_ = s.authClient.RevokeSessions(c.Request.Context(), bearerFromCtx(c), *p.AuthAccountID)
	}

	auditSN := s.audit.Next()
	_ = s.moderation.Record(c.Request.Context(), actorID, moderation.TargetPlayer, playerID, moderation.ActionKick, req.Reason, map[string]interface{}{
		"audit_sn": auditSN,
	})
	_ = opskick.Publish(c.Request.Context(), s.rdb, playerID, req.Reason, auditSN)

	c.JSON(http.StatusOK, gin.H{
		"player_id": playerID, "kicked": true, "audit_sn": auditSN, "request_id": c.GetString("request_id"),
	})
}

func (s *Server) disableAdminUser(c *gin.Context) {
	s.setAdminEnabled(c, false)
}

func (s *Server) enableAdminUser(c *gin.Context) {
	s.setAdminEnabled(c, true)
}

func (s *Server) setAdminEnabled(c *gin.Context, enable bool) {
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
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "cannot change own status", "request_id": c.GetString("request_id")})
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

	status := "disabled"
	action := moderation.ActionDisable
	if enable {
		status = "active"
		action = moderation.ActionEnable
	}

	if u.AuthAccountID != nil && s.authClient != nil {
		token := bearerFromCtx(c)
		if err := s.authClient.SetAccountStatus(c.Request.Context(), token, *u.AuthAccountID, status, req.Reason); err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"code": 502, "message": "auth status: " + err.Error(), "request_id": c.GetString("request_id")})
			return
		}
		if !enable {
			_ = s.authClient.RevokeSessions(c.Request.Context(), token, *u.AuthAccountID)
		}
	}

	if err := s.admins.SetStatus(c.Request.Context(), userID, status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}

	_ = s.moderation.Record(c.Request.Context(), actorID, moderation.TargetAdmin, userID, action, req.Reason, nil)
	c.JSON(http.StatusOK, gin.H{"user_id": userID, "status": status, "request_id": c.GetString("request_id")})
}

func (s *Server) patchAdminRole(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "cannot change own role", "request_id": c.GetString("request_id")})
		return
	}
	var req struct {
		Role   string `json:"role"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Role == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "role required", "request_id": c.GetString("request_id")})
		return
	}
	switch req.Role {
	case user.RolePlatformAdmin, user.RoleClubAdmin, user.RoleAgent:
	default:
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid role", "request_id": c.GetString("request_id")})
		return
	}
	u, err := s.admins.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "admin not found", "request_id": c.GetString("request_id")})
		return
	}
	if u.AuthAccountID != nil && s.authClient != nil {
		if err := s.authClient.SetAccountRole(c.Request.Context(), bearerFromCtx(c), *u.AuthAccountID, req.Role); err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"code": 502, "message": "auth role: " + err.Error(), "request_id": c.GetString("request_id")})
			return
		}
	}
	if err := s.admins.SetRole(c.Request.Context(), userID, req.Role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	_ = s.moderation.Record(c.Request.Context(), actorID, moderation.TargetAdmin, userID, moderation.ActionSetRole, req.Reason, map[string]interface{}{
		"role": req.Role,
	})
	c.JSON(http.StatusOK, gin.H{"user_id": userID, "role": req.Role, "request_id": c.GetString("request_id")})
}
