package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/example/game/internal/platform/lobby"
)

func (s *Server) listLobbyGames(c *gin.Context) {
	uid, err := s.playerID(c)
	if err != nil {
		return
	}
	games, err := s.lobby.ListLobbyGames(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"games": games})
}

func (s *Server) updateLobbyGames(c *gin.Context) {
	uid, err := s.playerID(c)
	if err != nil {
		return
	}
	var req struct {
		Games []lobby.GamePrefUpdate `json:"games"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	if err := s.lobby.UpdateLobbyGames(c.Request.Context(), uid, req.Games); err != nil {
		if err == lobby.ErrInvalidGame {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error(), "request_id": c.GetString("request_id")})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	games, err := s.lobby.ListLobbyGames(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"games": games})
}
