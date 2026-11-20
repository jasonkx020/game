package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GET /v1/discovery/assign?exclude_id=
func (s *Server) assignConnector(c *gin.Context) {
	uid, err := s.playerID(c)
	if err != nil {
		return
	}
	exclude := c.Query("exclude_id")
	var res interface{}
	if s.discovery != nil {
		a := s.discovery.AssignForUser(uid, exclude)
		body := gin.H{
			"ws_gateway_url":   a.WSGatewayURL,
			"http_gateway_url": a.HTTPGatewayURL,
		}
		if a.Connector != nil {
			conn := gin.H{
				"id":     a.Connector.ID,
				"ws_url": a.Connector.WSURL,
			}
			if a.Connector.WSDirectURL != "" {
				conn["ws_direct_url"] = a.Connector.WSDirectURL
			}
			body["connector"] = conn
		}
		res = body
	} else {
		res = gin.H{
			"ws_gateway_url":   s.cfg.WSGatewayURL,
			"http_gateway_url": s.cfg.HTTPGatewayURL,
		}
	}
	c.JSON(http.StatusOK, res)
}
