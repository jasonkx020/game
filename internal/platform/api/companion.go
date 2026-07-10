package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/example/game/internal/platform/companion"
)

func (s *Server) createCompanionSession(c *gin.Context) {
	uid, err := s.userID(c)
	if err != nil {
		return
	}
	var req struct {
		PersonaID string `json:"persona_id"`
	}
	_ = c.ShouldBindJSON(&req)
	sess, err := s.companion.CreateSession(c.Request.Context(), uid, req.PersonaID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusCreated, sess)
}

func (s *Server) listCompanionMessages(c *gin.Context) {
	uid, err := s.userID(c)
	if err != nil {
		return
	}
	sid, err := strconv.ParseInt(c.Param("session_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid session_id", "request_id": c.GetString("request_id")})
		return
	}
	msgs, err := s.companion.ListMessages(c.Request.Context(), sid, uid, 50)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"messages": msgs})
}

func (s *Server) companionChat(c *gin.Context) {
	uid, err := s.userID(c)
	if err != nil {
		return
	}
	sid, err := strconv.ParseInt(c.Param("session_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid session_id", "request_id": c.GetString("request_id")})
		return
	}
	var req struct {
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "message required", "request_id": c.GetString("request_id")})
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	if err := s.companion.ChatStream(c.Request.Context(), sid, uid, req.Message, c.Writer); err != nil {
		if err == companion.ErrSessionNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": err.Error(), "request_id": c.GetString("request_id")})
			return
		}
		_, _ = c.Writer.Write([]byte("data: {\"content\":\"抱歉，我这边出了点状况，稍后再聊～\"}\n\n"))
		_, _ = c.Writer.Write([]byte("data: [DONE]\n\n"))
	}
}

func (s *Server) listCompanionPersonas(c *gin.Context) {
	list, err := s.companion.ListPersonas(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"personas": list})
}

func (s *Server) lobbyRecommendations(c *gin.Context) {
	uid, err := s.userID(c)
	if err != nil {
		return
	}
	recs, err := s.companion.Recommendations(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"recommendations": recs})
}

func (s *Server) getUserSettings(c *gin.Context) {
	uid, err := s.userID(c)
	if err != nil {
		return
	}
	settings, err := s.users.GetSettings(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"settings": settings})
}

func (s *Server) putUserSettings(c *gin.Context) {
	uid, err := s.userID(c)
	if err != nil {
		return
	}
	var req struct {
		Settings map[string]interface{} `json:"settings"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	settings, err := s.users.UpdateSettings(c.Request.Context(), uid, req.Settings)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"settings": settings})
}
