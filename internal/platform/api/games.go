package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/example/game/internal/platform/catalog"
)

func (s *Server) listGames(c *gin.Context) {
	games, err := s.catalog.ListGames(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"games": games})
}

func (s *Server) getGameConfig(c *gin.Context) {
	gameID := c.Param("game_id")
	cfg, err := s.catalog.GetGameConfig(c.Request.Context(), gameID)
	if err != nil {
		if err == catalog.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "game not found", "request_id": c.GetString("request_id")})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	c.JSON(http.StatusOK, cfg)
}
