package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/example/game/pkg/platform/catalog"
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

func (s *Server) listCompanionPersonas(c *gin.Context) {
	var rows []struct {
		PersonaID string `db:"persona_id"`
		Name      string `db:"name"`
		AvatarURL string `db:"avatar_url"`
	}
	err := s.db.SelectContext(c.Request.Context(), &rows,
		`SELECT persona_id, name, avatar_url FROM companion_persona WHERE enabled ORDER BY persona_id`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error(), "request_id": c.GetString("request_id")})
		return
	}
	out := make([]map[string]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, map[string]string{
			"persona_id": r.PersonaID, "name": r.Name, "avatar_url": r.AvatarURL,
		})
	}
	c.JSON(http.StatusOK, gin.H{"personas": out})
}
