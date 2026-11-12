package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/example/game/internal/platform/club"
)

func (s *Server) createClub(c *gin.Context) {
	adminID, err := s.adminID(c)
	if err != nil {
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "name required", "request_id": c.GetString("request_id")})
		return
	}
	cl, err := s.clubs.Create(c.Request.Context(), adminID, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusCreated, cl)
}

func (s *Server) getClub(c *gin.Context) {
	clubID, err := parseClubID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid club_id", "request_id": c.GetString("request_id")})
		return
	}
	cl, err := s.clubs.Get(c.Request.Context(), clubID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	bal, _ := s.clubs.PoolBalance(c.Request.Context(), clubID)
	c.JSON(http.StatusOK, gin.H{
		"id": cl.ID, "name": cl.Name, "owner_admin_id": cl.OwnerAdminID,
		"status": cl.Status, "pool_balance": bal,
	})
}

func (s *Server) listClubMembers(c *gin.Context) {
	clubID, err := parseClubID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid club_id", "request_id": c.GetString("request_id")})
		return
	}
	members, err := s.clubs.ListMembers(c.Request.Context(), clubID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"members": members})
}

func (s *Server) addClubMember(c *gin.Context) {
	adminID, err := s.adminID(c)
	if err != nil {
		return
	}
	clubID, err := parseClubID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid club_id", "request_id": c.GetString("request_id")})
		return
	}
	var req struct {
		UserID int64 `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.UserID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "user_id required", "request_id": c.GetString("request_id")})
		return
	}
	if err := s.clubs.RequireClubOwner(c.Request.Context(), clubID, adminID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	if err := s.clubs.AddMember(c.Request.Context(), clubID, req.UserID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) removeClubMember(c *gin.Context) {
	adminID, err := s.adminID(c)
	if err != nil {
		return
	}
	clubID, err := parseClubID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid club_id", "request_id": c.GetString("request_id")})
		return
	}
	targetID, err := parseTargetUserID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid user_id", "request_id": c.GetString("request_id")})
		return
	}
	if err := s.clubs.RemoveMember(c.Request.Context(), clubID, targetID, adminID); err != nil {
		if err == club.ErrForbidden {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": err.Error(), "request_id": c.GetString("request_id")})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) clubPoolBalance(c *gin.Context) {
	clubID, err := parseClubID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid club_id", "request_id": c.GetString("request_id")})
		return
	}
	bal, err := s.clubs.PoolBalance(c.Request.Context(), clubID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"club_id": clubID, "balance": bal})
}

func (s *Server) transferToClubPool(c *gin.Context) {
	adminID, err := s.adminID(c)
	if err != nil {
		return
	}
	clubID, err := parseClubID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid club_id", "request_id": c.GetString("request_id")})
		return
	}
	var req struct {
		Amount   int64 `json:"amount"`
		PlayerID int64 `json:"player_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Amount <= 0 || req.PlayerID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "amount and player_id required", "request_id": c.GetString("request_id")})
		return
	}
	auditSN := s.audit.Next()
	bal, err := s.clubs.TransferToPool(c.Request.Context(), clubID, adminID, req.PlayerID, req.Amount, auditSN)
	if err != nil {
		if err == club.ErrForbidden || err == club.ErrInsufficientWallet {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"club_id": clubID, "pool_balance": bal, "audit_sn": auditSN})
}

func (s *Server) adminListClubs(c *gin.Context) {
	clubs, err := s.clubs.ListAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"clubs": clubs})
}
