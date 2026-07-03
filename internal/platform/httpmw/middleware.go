package httpmw

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
)

const emptyBodySHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

type SignatureConfig struct {
	AppID     string
	AppSecret string
	Skip      bool
}

func Signature(cfg SignatureConfig, rdb *goredis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg.Skip {
			c.Next()
			return
		}
		appID := c.GetHeader("X-App-Id")
		tsStr := c.GetHeader("X-Timestamp")
		nonce := c.GetHeader("X-Nonce")
		contentSHA := c.GetHeader("X-Content-SHA256")
		sig := c.GetHeader("X-Signature")

		if appID != cfg.AppID || tsStr == "" || nonce == "" || contentSHA == "" || sig == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 1002, "message": "signature headers missing", "request_id": c.GetString("request_id")})
			return
		}
		ts, err := strconv.ParseInt(tsStr, 10, 64)
		if err != nil || abs(time.Now().Unix()-ts) > 300 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 1003, "message": "timestamp expired", "request_id": c.GetString("request_id")})
			return
		}
		if rdb != nil {
			key := "nonce:" + nonce
			ok, err := rdb.SetNX(c.Request.Context(), key, 1, 10*time.Minute).Result()
			if err != nil || !ok {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 1004, "message": "nonce replay", "request_id": c.GetString("request_id")})
				return
			}
		}

		body, _ := io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		hash := sha256.Sum256(body)
		actualSHA := hex.EncodeToString(hash[:])
		if actualSHA != contentSHA {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 1002, "message": "content sha256 mismatch", "request_id": c.GetString("request_id")})
			return
		}

		auth := c.GetHeader("Authorization")
		canonical := strings.Join([]string{
			c.Request.Method,
			c.Request.URL.Path,
			canonicalQuery(c.Request.URL.Query()),
			tsStr,
			nonce,
			contentSHA,
			auth,
		}, "\n")
		mac := hmac.New(sha256.New, []byte(cfg.AppSecret))
		mac.Write([]byte(canonical))
		expected := hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(expected), []byte(sig)) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 1002, "message": "signature invalid", "request_id": c.GetString("request_id")})
			return
		}
		c.Next()
	}
}

func JWT(secret string) gin.HandlerFunc {
	sec := []byte(secret)
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 1001, "message": "token required", "request_id": c.GetString("request_id")})
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		c.Set("access_token", token)
		_ = sec
		c.Next()
	}
}

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("request_id", fmt.Sprintf("%d", time.Now().UnixNano()))
		c.Next()
	}
}

func canonicalQuery(q url.Values) string {
	if len(q) == 0 {
		return ""
	}
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		for _, v := range q[k] {
			parts = append(parts, k+"="+url.QueryEscape(v))
		}
	}
	sort.Strings(parts)
	return strings.Join(parts, "&")
}

func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
