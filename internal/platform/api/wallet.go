package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) roomCardBalance(c *gin.Context) {
	uid, err := s.playerID(c)
	if err != nil {
		return
	}
	bal, err := s.wallet.RoomCardBalance(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"balance": bal})
}

func (s *Server) gameCoinBalance(c *gin.Context) {
	uid, err := s.playerID(c)
	if err != nil {
		return
	}
	gameID := c.Param("game_id")
	bal, err := s.wallet.GameCoinBalance(c.Request.Context(), uid, gameID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"balance": bal, "game_id": gameID, "coin_name": "龟币"})
}

func (s *Server) mockRecharge(c *gin.Context) {
	uid, err := s.playerID(c)
	if err != nil {
		return
	}
	var req struct {
		ProductID string `json:"product_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	auditSN := s.audit.Next()
	order, bal, err := s.wallet.MockRecharge(c.Request.Context(), uid, req.ProductID, auditSN)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"order": order, "balance": bal, "audit_sn": auditSN,
	})
}

func (s *Server) rechargeHistory(c *gin.Context) {
	uid, err := s.playerID(c)
	if err != nil {
		return
	}
	orders, err := s.wallet.RechargeHistory(c.Request.Context(), uid, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"orders": orders})
}
