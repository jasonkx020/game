package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/example/game/internal/platform/player"
	"github.com/example/game/internal/platform/user"
)

func (s *Server) login(c *gin.Context) {
	var req struct {
		Phone   string `json:"phone"`
		SMSCode string `json:"sms_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	p, token, err := s.players.Login(c.Request.Context(), req.Phone, req.SMSCode)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1001, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"access_token":  token,
		"refresh_token": token,
		"expires_in":    86400,
		"user_id":       p.ID,
		"nickname":      p.Nickname,
		"role":          user.RolePlayer,
	})
}

func (s *Server) refresh(c *gin.Context) {
	uid, err := s.playerID(c)
	if err != nil {
		return
	}
	newToken, err := s.players.Refresh(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"access_token": newToken, "expires_in": 86400})
}

func (s *Server) profile(c *gin.Context) {
	uid, err := s.playerID(c)
	if err != nil {
		return
	}
	p, err := s.players.GetByID(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "user not found", "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"user_id":      p.ID,
		"phone":        p.Phone,
		"phone_masked": user.MaskPhone(p.Phone),
		"nickname":     p.Nickname,
		"role":         user.RolePlayer,
		"avatar_url":   p.AvatarURL,
	})
}

func (s *Server) putProfile(c *gin.Context) {
	uid, err := s.playerID(c)
	if err != nil {
		return
	}
	var req struct {
		Nickname  *string `json:"nickname"`
		AvatarURL *string `json:"avatar_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	if req.Nickname == nil && req.AvatarURL == nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "nickname or avatar_url required", "request_id": c.GetString("request_id")})
		return
	}
	if req.Nickname != nil {
		n := *req.Nickname
		if len(n) == 0 || len(n) > 64 {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "nickname length invalid", "request_id": c.GetString("request_id")})
			return
		}
	}
	p, err := s.players.UpdateProfile(c.Request.Context(), uid, player.UpdateProfileParams{
		Nickname:  req.Nickname,
		AvatarURL: req.AvatarURL,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"user_id":      p.ID,
		"phone":        p.Phone,
		"phone_masked": user.MaskPhone(p.Phone),
		"nickname":     p.Nickname,
		"role":         user.RolePlayer,
		"avatar_url":   p.AvatarURL,
	})
}
