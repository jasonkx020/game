package config

import (
	"crypto/ecdsa"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"

	"github.com/example/game/pkg/platform/user"
)

type Config struct {
	DatabaseURL       string
	RedisURL          string
	HMACSecret        string
	AppID             string
	DevSMSCode        string
	ListenHost        string // bind address; default 0.0.0.0
	GameAPIHTTPPort   string
	AdminHTTPPort     string
	PitayaWSPort      int
	PitayaWSURL       string
	SnowflakeWorkerID int64
	CORSOrigins       []string
	LLMBaseURL        string
	LLMAPIKey         string
	LLMModel          string
	LLMTimeoutSec     int
	AuthBaseURL       string

	JWTPublicKeyPEM  string
	JWTPublicKeyFile string
	JWTPublicKey     *ecdsa.PublicKey

	RiskGPSHardBlock bool
	RiskAutoFreeze   bool

	// Connector discovery (ADR-009): cmd/game publishes; game-api assigns.
	DiscoveryEnabled     bool
	ConnectorInstanceID  string
	ConnectorWSDirectURL string
	WSGatewayURL         string
	HTTPGatewayURL       string
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
	v.SetDefault("HTTP_LISTEN_HOST", "0.0.0.0")
	v.SetDefault("ADMIN_HTTP_PORT", "8082")
	v.SetDefault("PITAYA_WS_PORT", 3250)
	v.SetDefault("PITAYA_WS_URL", "ws://localhost:3250")
	v.SetDefault("SNOWFLAKE_WORKER_ID", 1)
	v.SetDefault("CORS_ORIGINS", "http://localhost:8082,http://localhost:5173,http://localhost:7456,http://127.0.0.1:7456")
	v.SetDefault("LLM_BASE_URL", "https://api.openai.com/v1")
	v.SetDefault("LLM_MODEL", "gpt-4o-mini")
	v.SetDefault("LLM_TIMEOUT_SEC", 60)
	v.SetDefault("AUTH_BASE_URL", "http://localhost:8081")
	v.SetDefault("JWT_PUBLIC_KEY_FILE", "keys/jwt-es256-dev.pub.pem")
	v.SetDefault("RISK_GPS_HARD_BLOCK", false)
	v.SetDefault("RISK_AUTO_FREEZE", false)
	v.SetDefault("DISCOVERY_ENABLED", true)

	if envPath := findEnvFile(); envPath != "" {
		v.SetConfigFile(envPath)
		_ = v.ReadInConfig()
	}

	corsRaw := v.GetString("CORS_ORIGINS")
	var corsOrigins []string
	for _, o := range strings.Split(corsRaw, ",") {
		if t := strings.TrimSpace(o); t != "" {
			corsOrigins = append(corsOrigins, t)
		}
	}

	cfg := &Config{
		DatabaseURL:          v.GetString("DATABASE_URL"),
		RedisURL:             v.GetString("REDIS_URL"),
		HMACSecret:           v.GetString("HMAC_SECRET"),
		AppID:                v.GetString("APP_ID"),
		DevSMSCode:           v.GetString("DEV_SMS_CODE"),
		ListenHost:           firstNonEmpty(v.GetString("HTTP_LISTEN_HOST"), "0.0.0.0"),
		GameAPIHTTPPort:      firstNonEmpty(v.GetString("GAME_API_HTTP_PORT"), v.GetString("PLATFORM_HTTP_PORT"), "8080"),
		AdminHTTPPort:        firstNonEmpty(v.GetString("ADMIN_HTTP_PORT"), "8082"),
		PitayaWSPort:         v.GetInt("PITAYA_WS_PORT"),
		PitayaWSURL:          v.GetString("PITAYA_WS_URL"),
		SnowflakeWorkerID:    v.GetInt64("SNOWFLAKE_WORKER_ID"),
		CORSOrigins:          corsOrigins,
		LLMBaseURL:           v.GetString("LLM_BASE_URL"),
		LLMAPIKey:            v.GetString("LLM_API_KEY"),
		LLMModel:             v.GetString("LLM_MODEL"),
		LLMTimeoutSec:        v.GetInt("LLM_TIMEOUT_SEC"),
		AuthBaseURL:          firstNonEmpty(v.GetString("AUTH_BASE_URL"), "http://localhost:8081"),
		JWTPublicKeyPEM:      v.GetString("JWT_PUBLIC_KEY_PEM"),
		JWTPublicKeyFile:     resolveKeyPath(v.GetString("JWT_PUBLIC_KEY_FILE")),
		RiskGPSHardBlock:     v.GetBool("RISK_GPS_HARD_BLOCK"),
		RiskAutoFreeze:       v.GetBool("RISK_AUTO_FREEZE"),
		DiscoveryEnabled:     v.GetBool("DISCOVERY_ENABLED"),
		ConnectorInstanceID:  strings.TrimSpace(v.GetString("CONNECTOR_INSTANCE_ID")),
		ConnectorWSDirectURL: strings.TrimSpace(v.GetString("CONNECTOR_WS_DIRECT_URL")),
		WSGatewayURL:         strings.TrimSpace(v.GetString("WS_GATEWAY_URL")),
		HTTPGatewayURL:       strings.TrimSpace(v.GetString("HTTP_GATEWAY_URL")),
	}
	if cfg.ConnectorInstanceID == "" {
		cfg.ConnectorInstanceID = "game-" + strings.TrimPrefix(cfg.PitayaWSURL, "ws://")
		cfg.ConnectorInstanceID = strings.ReplaceAll(cfg.ConnectorInstanceID, "/", "-")
	}
	if cfg.WSGatewayURL == "" {
		cfg.WSGatewayURL = cfg.PitayaWSURL
	}

	pemBytes, err := user.ResolvePEM(cfg.JWTPublicKeyPEM, cfg.JWTPublicKeyFile)
	if err != nil {
		return nil, fmt.Errorf("JWT public key: %w", err)
	}
	pub, err := user.LoadPublicKeyPEM(pemBytes)
	if err != nil {
		return nil, fmt.Errorf("JWT public key: %w", err)
	}
	cfg.JWTPublicKey = pub
	return cfg, nil
}

func firstNonEmpty(vals ...string) string {
	for _, s := range vals {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

// ListenAddr builds host:port for HTTP/WS bind (empty host → 0.0.0.0).
func ListenAddr(host string, port string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		host = "0.0.0.0"
	}
	port = strings.TrimPrefix(strings.TrimSpace(port), ":")
	return host + ":" + port
}

func (c *Config) GameAPIAddr() string {
	return ListenAddr(c.ListenHost, c.GameAPIHTTPPort)
}

func (c *Config) AdminAPIAddr() string {
	return ListenAddr(c.ListenHost, c.AdminHTTPPort)
}

func (c *Config) PitayaWSAddr() string {
	return ListenAddr(c.ListenHost, fmt.Sprintf("%d", c.PitayaWSPort))
}

func resolveKeyPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" || filepath.IsAbs(p) {
		return p
	}
	root := findModuleRoot()
	candidates := []string{p}
	if root != "" {
		candidates = append(candidates,
			filepath.Join(root, p),
			filepath.Join(root, "server", p),
			filepath.Join(root, "..", "server", p),
			filepath.Join(root, "..", "auth-service", "keys", filepath.Base(p)),
			filepath.Join(root, "keys", filepath.Base(p)),
		)
	}
	for _, cand := range candidates {
		if _, err := os.Stat(cand); err == nil {
			abs, err := filepath.Abs(cand)
			if err == nil {
				return abs
			}
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
		envPath := filepath.Join(dir, ".env")
		if _, err := os.Stat(envPath); err == nil {
			return envPath
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
