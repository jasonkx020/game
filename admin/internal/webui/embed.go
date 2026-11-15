package webui

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed all:dist
var Dist embed.FS

// Mount registers embedded SPA static files and history-mode fallback.
// Call after API routes. Paths under /v1 are never rewritten to index.html.
func Mount(r *gin.Engine) {
	sub, err := fs.Sub(Dist, "dist")
	if err != nil {
		panic("webui: dist subfs: " + err.Error())
	}
	fileServer := http.FileServer(http.FS(sub))

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/v1" || strings.HasPrefix(path, "/v1/") {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
				"code": http.StatusNotFound, "message": "not found", "request_id": c.GetString("request_id"),
			})
			return
		}

		name := strings.TrimPrefix(path, "/")
		if name == "" {
			name = "index.html"
		}
		if f, err := sub.Open(name); err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}

		// Vue Router history fallback
		data, err := Dist.ReadFile("dist/index.html")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"code": 500, "message": "admin ui missing; run npm run build in admin/web",
			})
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})
}
