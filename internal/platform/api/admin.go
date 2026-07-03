package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (s *Server) adminMetricsOverview(c *gin.Context) {
	o, err := s.metrics.Overview(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, o)
}

func (s *Server) adminMetricsRoomCards(c *gin.Context) {
	days := 7
	if d := c.Query("days"); d != "" {
		if n, err := strconv.Atoi(d); err == nil {
			days = n
		}
	}
	trend, err := s.metrics.RoomCardTrend(c.Request.Context(), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"days": trend})
}
