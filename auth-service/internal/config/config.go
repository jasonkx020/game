package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/example/auth-service/internal/token"
)

type Config struct {
	DatabaseURL        string
	RedisURL           string
	HMACSecret         string
	AppID              string
	DevSMSCode         string
	DevEmailCode       string
	AuthHTTPPort       string
	AccessTokenExpire  time.Duration
	RefreshTokenExpire time.Duration
	RefreshGrace       time.Duration
	CORSOrigins        []string
	AdminDevPhone      string

	JWTPrivateKeyPEM string
	JWTPublicKeyPEM  string
	JWTPrivateKeyFile string
	JWTPublicKeyFile  string
	JWTKeyID          string

	WeChatMode      string
	WeChatMPAppID   string
	WeChatMPSecret  string
	WeChatWebAppID  string
	WeChatWebSecret string

	KYCMode                 string
	AliyunAccessKeyID       string
	AliyunAccessKeySecret   string
	AliyunCloudauthEndpoint string
}

func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigType("env")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	v.SetDefault("DATABASE_URL", "postgres://game:game@localhost:5432/game?sslmode=disable")
	v.SetDefault("REDIS_URL", "redis://localhost:6379/0")
	v.SetDefault("HMAC_SECRET", "dev-hmac-secret")
	v.SetDefault("APP_ID", "cocos-dev")
	v.SetDefault("DEV_SMS_CODE", "123456")
	v.SetDefault("DEV_EMAIL_CODE", "123456")
	v.SetDefault("AUTH_HTTP_PORT", "8081")
	v.SetDefault("ACCESS_TOKEN_EXPIRE_MINUTES", 120)
	v.SetDefault("REFRESH_TOKEN_EXPIRE_DAYS", 90)
	v.SetDefault("REFRESH_GRACE_SECONDS", 120)
	v.SetDefault("CORS_ORIGINS", "http://localhost:5173,http://localhost:7456,http://127.0.0.1:7456")
	v.SetDefault("ADMIN_DEV_PHONE", "13800000000")
	v.SetDefault("JWT_KEY_ID", "es256-1")
	v.SetDefault("JWT_PRIVATE_KEY_FILE", "keys/jwt-es256-dev.pem")
	v.SetDefault("JWT_PUBLIC_KEY_FILE", "keys/jwt-es256-dev.pub.pem")
	v.SetDefault("WECHAT_MODE", "mock")
	v.SetDefault("KYC_MODE", "mock")
	v.SetDefault("ALIYUN_CLOUDAUTH_ENDPOINT", "cloudauth.aliyuncs.com")

	if envPath := findEnvFile(); envPath != "" {
		v.SetConfigFile(envPath)
		_ = v.ReadInConfig()
	}

	var cors []string
	for _, o := range strings.Split(v.GetString("CORS_ORIGINS"), ",") {
		if t := strings.TrimSpace(o); t != "" {
			cors = append(cors, t)
		}
	}

	accessMin := v.GetInt("ACCESS_TOKEN_EXPIRE_MINUTES")
	if accessMin <= 0 {
		accessMin = 120
	}
	refreshDays := v.GetInt("REFRESH_TOKEN_EXPIRE_DAYS")
	if refreshDays <= 0 {
		refreshDays = 90
	}
	graceSec := v.GetInt("REFRESH_GRACE_SECONDS")
	if graceSec <= 0 {
		graceSec = 120
	}

	cfg := &Config{
		DatabaseURL:             v.GetString("DATABASE_URL"),
		RedisURL:                v.GetString("REDIS_URL"),
		HMACSecret:              v.GetString("HMAC_SECRET"),
		AppID:                   v.GetString("APP_ID"),
		DevSMSCode:              v.GetString("DEV_SMS_CODE"),
		DevEmailCode:            v.GetString("DEV_EMAIL_CODE"),
		AuthHTTPPort:            v.GetString("AUTH_HTTP_PORT"),
		AccessTokenExpire:       time.Duration(accessMin) * time.Minute,
		RefreshTokenExpire:      time.Duration(refreshDays) * 24 * time.Hour,
		RefreshGrace:            time.Duration(graceSec) * time.Second,
		CORSOrigins:             cors,
		AdminDevPhone:           v.GetString("ADMIN_DEV_PHONE"),
		JWTPrivateKeyPEM:        v.GetString("JWT_PRIVATE_KEY_PEM"),
		JWTPublicKeyPEM:         v.GetString("JWT_PUBLIC_KEY_PEM"),
		JWTPrivateKeyFile:       resolveKeyPath(v.GetString("JWT_PRIVATE_KEY_FILE")),
		JWTPublicKeyFile:        resolveKeyPath(v.GetString("JWT_PUBLIC_KEY_FILE")),
		JWTKeyID:                v.GetString("JWT_KEY_ID"),
		WeChatMode:              v.GetString("WECHAT_MODE"),
		WeChatMPAppID:           v.GetString("WECHAT_MP_APP_ID"),
		WeChatMPSecret:          v.GetString("WECHAT_MP_SECRET"),
		WeChatWebAppID:          v.GetString("WECHAT_WEB_APP_ID"),
		WeChatWebSecret:         v.GetString("WECHAT_WEB_SECRET"),
		KYCMode:                 v.GetString("KYC_MODE"),
		AliyunAccessKeyID:       v.GetString("ALIYUN_ACCESS_KEY_ID"),
		AliyunAccessKeySecret:   v.GetString("ALIYUN_ACCESS_KEY_SECRET"),
		AliyunCloudauthEndpoint: v.GetString("ALIYUN_CLOUDAUTH_ENDPOINT"),
	}
	return cfg, nil
}

// LoadTokenManager loads ES256 keys and builds a signing Manager.
func (c *Config) LoadTokenManager() (*token.Manager, error) {
	privPEM, err := token.ResolvePEM(c.JWTPrivateKeyPEM, c.JWTPrivateKeyFile)
	if err != nil {
		return nil, fmt.Errorf("JWT private key: %w", err)
	}
	priv, err := token.LoadPrivateKeyPEM(privPEM)
	if err != nil {
		return nil, fmt.Errorf("JWT private key: %w", err)
	}
	// Optional public override for consistency checks
	if c.JWTPublicKeyPEM != "" || c.JWTPublicKeyFile != "" {
		pubPEM, err := token.ResolvePEM(c.JWTPublicKeyPEM, c.JWTPublicKeyFile)
		if err != nil {
			return nil, fmt.Errorf("JWT public key: %w", err)
		}
		pub, err := token.LoadPublicKeyPEM(pubPEM)
		if err != nil {
			return nil, fmt.Errorf("JWT public key: %w", err)
		}
		if pub.X.Cmp(priv.PublicKey.X) != 0 || pub.Y.Cmp(priv.PublicKey.Y) != 0 {
			return nil, fmt.Errorf("JWT public key does not match private key")
		}
	}
	return token.NewManager(priv, c.JWTKeyID, c.AccessTokenExpire), nil
}

func resolveKeyPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" || filepath.IsAbs(p) {
		return p
	}
	// Prefer path relative to auth-service module (go.mod)
	if root := findModuleRoot(); root != "" {
		cand := filepath.Join(root, p)
		if _, err := os.Stat(cand); err == nil {
			return cand
		}
	}
	return p
}

func findModuleRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func findEnvFile() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			envPath := filepath.Join(dir, ".env")
			if _, err := os.Stat(envPath); err == nil {
				return envPath
			}
			return ""
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
