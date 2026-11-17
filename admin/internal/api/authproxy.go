package api

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// mountAuthProxy forwards /v1/admin/auth* and /v1/auth* to auth-service so the
// embedded SPA on :8082 can login without nginx gateway or VITE_AUTH_BASE.
// Paths are preserved (HMAC signs the full /v1/... path).
func mountAuthProxy(r *gin.Engine, authBase string) {
	authBase = strings.TrimRight(strings.TrimSpace(authBase), "/")
	if authBase == "" {
		return
	}
	target, err := url.Parse(authBase)
	if err != nil || target.Scheme == "" || target.Host == "" {
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, e error) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"code":502,"message":"auth-service unavailable"}`))
	}
	h := func(c *gin.Context) {
		proxy.ServeHTTP(c.Writer, c.Request)
		c.Abort()
	}
	r.Any("/v1/admin/auth", h)
	r.Any("/v1/admin/auth/*filepath", h)
	r.Any("/v1/auth", h)
	r.Any("/v1/auth/*filepath", h)
}
