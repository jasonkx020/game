package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

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
