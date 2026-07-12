package httpmw

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/example/game/internal/platform/user"
)

func CORS(origins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(origins))
	for _, o := range origins {
		allowed[strings.TrimSpace(o)] = struct{}{}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if _, ok := allowed[origin]; ok {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-App-Id, X-Timestamp, X-Nonce, X-Content-SHA256, X-Signature, Idempotency-Key")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func RequirePrincipalType(jwtSecret, want string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := bearerToken(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 1001, "message": "token required", "request_id": c.GetString("request_id")})
			return
		}
		typ, err := user.ParsePrincipalType(token, []byte(jwtSecret))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 1001, "message": "invalid token", "request_id": c.GetString("request_id")})
			return
		}
		if typ != want {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "message": "forbidden", "request_id": c.GetString("request_id")})
			return
		}
		c.Set("principal_type", typ)
		c.Next()
	}
}

func RequireRole(jwtSecret string, roles ...string) gin.HandlerFunc {
	roleSet := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		roleSet[r] = struct{}{}
	}
	return func(c *gin.Context) {
		token := bearerToken(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 1001, "message": "token required", "request_id": c.GetString("request_id")})
			return
		}
		role, err := user.ParseRole(token, []byte(jwtSecret))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 1001, "message": "invalid token", "request_id": c.GetString("request_id")})
			return
		}
		if _, ok := roleSet[role]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "message": "forbidden", "request_id": c.GetString("request_id")})
			return
		}
		c.Set("user_role", role)
		c.Next()
	}
}

func bearerToken(c *gin.Context) string {
	if t := c.GetString("access_token"); t != "" {
		return t
	}
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}
