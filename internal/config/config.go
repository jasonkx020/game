package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	DatabaseURL       string
	RedisURL          string
	JWTSecret         string
	HMACSecret        string
	AppID             string
	DevSMSCode        string
	PlatformHTTPPort  string
	PitayaWSPort      int
	PitayaWSURL       string
	SnowflakeWorkerID int64
	CORSOrigins       []string
	LLMBaseURL        string
	LLMAPIKey         string
	LLMModel          string
	LLMTimeoutSec     int
}

func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigType("env")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	v.SetDefault("DATABASE_URL", "postgres://game:game@localhost:5432/game?sslmode=disable")
	v.SetDefault("REDIS_URL", "redis://localhost:6379/0")
	v.SetDefault("JWT_SECRET", "dev-jwt-secret")
	v.SetDefault("HMAC_SECRET", "dev-hmac-secret")
	v.SetDefault("APP_ID", "cocos-dev")
	v.SetDefault("DEV_SMS_CODE", "123456")
	v.SetDefault("PLATFORM_HTTP_PORT", "8080")
	v.SetDefault("PITAYA_WS_PORT", 3250)
	v.SetDefault("PITAYA_WS_URL", "ws://localhost:3250")
	v.SetDefault("SNOWFLAKE_WORKER_ID", 1)
	v.SetDefault("CORS_ORIGINS", "http://localhost:5173")
	v.SetDefault("LLM_BASE_URL", "https://api.openai.com/v1")
	v.SetDefault("LLM_MODEL", "gpt-4o-mini")
	v.SetDefault("LLM_TIMEOUT_SEC", 60)

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

	return &Config{
		DatabaseURL:       v.GetString("DATABASE_URL"),
		RedisURL:          v.GetString("REDIS_URL"),
		JWTSecret:         v.GetString("JWT_SECRET"),
		HMACSecret:        v.GetString("HMAC_SECRET"),
		AppID:             v.GetString("APP_ID"),
		DevSMSCode:        v.GetString("DEV_SMS_CODE"),
		PlatformHTTPPort:  v.GetString("PLATFORM_HTTP_PORT"),
		PitayaWSPort:      v.GetInt("PITAYA_WS_PORT"),
		PitayaWSURL:       v.GetString("PITAYA_WS_URL"),
		SnowflakeWorkerID: v.GetInt64("SNOWFLAKE_WORKER_ID"),
		CORSOrigins:       corsOrigins,
		LLMBaseURL:        v.GetString("LLM_BASE_URL"),
		LLMAPIKey:         v.GetString("LLM_API_KEY"),
		LLMModel:          v.GetString("LLM_MODEL"),
		LLMTimeoutSec:     v.GetInt("LLM_TIMEOUT_SEC"),
	}, nil
}

// findEnvFile walks up from cwd to locate repo-root .env (go.mod sibling).
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
