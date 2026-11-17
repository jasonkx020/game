package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/example/game/pkg/platform/moderation"
	"github.com/example/game/pkg/platform/risk"
	"github.com/example/game/pkg/platform/user"
)

func (s *Server) listRiskAlerts(c *gin.Context) {
	if _, err := s.adminID(c); err != nil {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	typ := c.Query("type")
	status := c.Query("status")
	rows, total, err := s.risk.ListAlerts(c.Request.Context(), typ, status, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"alerts": rows, "total": total, "page": page, "page_size": pageSize})
}

func (s *Server) getRiskAlert(c *gin.Context) {
	if _, err := s.adminID(c); err != nil {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid id", "request_id": c.GetString("request_id")})
		return
	}
	a, err := s.risk.GetAlert(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, risk.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "not found", "request_id": c.GetString("request_id")})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, a)
}

func (s *Server) reviewRiskAlert(c *gin.Context) {
	actorID, err := s.adminID(c)
	if err != nil {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid id", "request_id": c.GetString("request_id")})
		return
	}
	var req struct {
		Decision string `json:"decision"` // dismissed | confirmed
		Note     string `json:"note"`
		Freeze   bool   `json:"freeze"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	if req.Decision != "dismissed" && req.Decision != "confirmed" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "decision must be dismissed|confirmed", "request_id": c.GetString("request_id")})
		return
	}
	if err := s.risk.ReviewAlert(c.Request.Context(), id, actorID, req.Decision, req.Note, req.Freeze); err != nil {
		if errors.Is(err, risk.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "not found", "request_id": c.GetString("request_id")})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	if req.Freeze && req.Decision == "confirmed" {
		if a, err := s.risk.GetAlert(c.Request.Context(), id); err == nil && a.PlayerID != nil {
			_ = s.moderation.Record(c.Request.Context(), actorID, moderation.TargetPlayer, *a.PlayerID,
				moderation.ActionRiskFreeze, req.Note, map[string]interface{}{"alert_id": id})
		}
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "decision": req.Decision, "request_id": c.GetString("request_id")})
}

func (s *Server) riskTableSnapshot(c *gin.Context) {
	if _, err := s.adminID(c); err != nil {
		return
	}
	roomID, err := uuid.Parse(c.Query("room_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid room_id", "request_id": c.GetString("request_id")})
		return
	}
	rows, err := s.risk.TableSnapshot(c.Request.Context(), roomID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"room_id": roomID.String(), "seats": rows})
}

func (s *Server) getAgentDeposit(c *gin.Context) {
	if _, err := s.adminID(c); err != nil {
		return
	}
	agentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || agentID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid id", "request_id": c.GetString("request_id")})
		return
	}
	bal, err := s.risk.AgentDepositBalance(c.Request.Context(), agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	rows, err := s.risk.ListAgentDeposits(c.Request.Context(), agentID, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"agent_admin_id": agentID,
		"required_cny":   risk.AgentDepositRequired,
		"balance_cny":    bal,
		"ledger":         rows,
	})
}

func (s *Server) postAgentDeposit(c *gin.Context) {
	actorID, err := s.adminID(c)
	if err != nil {
		return
	}
	agentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || agentID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid id", "request_id": c.GetString("request_id")})
		return
	}
	var req struct {
		Kind   string  `json:"kind"` // deposit | penalty
		Amount float64 `json:"amount_cny"`
		Reason string  `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	row, err := s.risk.RecordAgentDeposit(c.Request.Context(), agentID, req.Kind, req.Amount, req.Reason, &actorID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, row)
}

func (s *Server) listAgentWithdrawals(c *gin.Context) {
	if _, err := s.adminID(c); err != nil {
		return
	}
	agentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || agentID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid id", "request_id": c.GetString("request_id")})
		return
	}
	rows, err := s.risk.ListWithdrawals(c.Request.Context(), agentID, c.Query("status"), 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"withdrawals": rows})
}

func (s *Server) createAgentWithdrawal(c *gin.Context) {
	if _, err := s.adminID(c); err != nil {
		return
	}
	agentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || agentID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid id", "request_id": c.GetString("request_id")})
		return
	}
	var req struct {
		Amount float64 `json:"amount_cny"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	row, err := s.risk.RequestWithdrawal(c.Request.Context(), agentID, req.Amount, time.Now())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, row)
}

func (s *Server) reviewAgentWithdrawal(c *gin.Context) {
	actorID, err := s.adminID(c)
	if err != nil {
		return
	}
	agentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || agentID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid agent id", "request_id": c.GetString("request_id")})
		return
	}
	wid, err := strconv.ParseInt(c.Param("wid"), 10, 64)
	if err != nil || wid <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid withdrawal id", "request_id": c.GetString("request_id")})
		return
	}
	var req struct {
		Approve bool   `json:"approve"`
		Force   bool   `json:"force"`
		Note    string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	if req.Force {
		claims, err := s.accessClaims(c)
		if err != nil || claims.Role != user.RolePlatformAdmin {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "force approve requires platform_admin", "request_id": c.GetString("request_id")})
			return
		}
		if req.Note == "" {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "force approve requires note", "request_id": c.GetString("request_id")})
			return
		}
	}
	_ = agentID
	if err := s.risk.ReviewWithdrawal(c.Request.Context(), wid, actorID, req.Approve, req.Force, req.Note, time.Now()); err != nil {
		if errors.Is(err, risk.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "not found", "request_id": c.GetString("request_id")})
			return
		}
		if errors.Is(err, risk.ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": err.Error(), "request_id": c.GetString("request_id")})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": wid, "approve": req.Approve, "force": req.Force, "request_id": c.GetString("request_id")})
}
