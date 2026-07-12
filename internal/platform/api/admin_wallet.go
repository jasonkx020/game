package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (s *Server) adminRoomCardBalance(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Query("player_id"), 10, 64)
	if err != nil || playerID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "player_id required", "request_id": c.GetString("request_id")})
		return
	}
	bal, err := s.wallet.RoomCardBalance(c.Request.Context(), playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"balance": bal, "player_id": playerID})
}

func (s *Server) adminMockRecharge(c *gin.Context) {
	var req struct {
		PlayerID  int64  `json:"player_id"`
		ProductID string `json:"product_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.PlayerID <= 0 || req.ProductID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "player_id and product_id required", "request_id": c.GetString("request_id")})
		return
	}
	auditSN := s.audit.Next()
	order, bal, err := s.wallet.MockRecharge(c.Request.Context(), req.PlayerID, req.ProductID, auditSN)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"order": order, "balance": bal, "audit_sn": auditSN, "player_id": req.PlayerID,
	})
}

func (s *Server) adminRechargeHistory(c *gin.Context) {
	playerID, err := strconv.ParseInt(c.Query("player_id"), 10, 64)
	if err != nil || playerID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "player_id required", "request_id": c.GetString("request_id")})
		return
	}
	orders, err := s.wallet.RechargeHistory(c.Request.Context(), playerID, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"orders": orders, "player_id": playerID})
}
