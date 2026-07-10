package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

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
	u, token, err := s.users.Login(c.Request.Context(), req.Phone, req.SMSCode)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1001, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"access_token":  token,
		"refresh_token": token,
		"expires_in":    86400,
		"user_id":       u.ID,
		"nickname":      u.Nickname,
		"role":          u.Role,
	})
}

func (s *Server) refresh(c *gin.Context) {
	uid, err := s.userID(c)
	if err != nil {
		return
	}
	newToken, err := s.users.Refresh(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"access_token": newToken, "expires_in": 86400})
}

func (s *Server) profile(c *gin.Context) {
	uid, err := s.userID(c)
	if err != nil {
		return
	}
	u, err := s.users.GetByID(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "user not found", "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"user_id":      u.ID,
		"phone":        u.Phone,
		"phone_masked": user.MaskPhone(u.Phone),
		"nickname":     u.Nickname,
		"role":         u.Role,
		"avatar_url":   u.AvatarURL,
	})
}
