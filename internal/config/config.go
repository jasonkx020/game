package config

import (
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
}

func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	_ = viper.ReadInConfig()
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	viper.SetDefault("DATABASE_URL", "postgres://game:game@localhost:5432/game?sslmode=disable")
	viper.SetDefault("REDIS_URL", "redis://localhost:6379/0")
	viper.SetDefault("JWT_SECRET", "dev-jwt-secret")
	viper.SetDefault("HMAC_SECRET", "dev-hmac-secret")
	viper.SetDefault("APP_ID", "cocos-dev")
	viper.SetDefault("DEV_SMS_CODE", "123456")
	viper.SetDefault("PLATFORM_HTTP_PORT", "8080")
	viper.SetDefault("PITAYA_WS_PORT", 3250)
	viper.SetDefault("PITAYA_WS_URL", "ws://localhost:3250")
	viper.SetDefault("SNOWFLAKE_WORKER_ID", 1)
	viper.SetDefault("CORS_ORIGINS", "http://localhost:5173")

	corsRaw := viper.GetString("CORS_ORIGINS")
	var corsOrigins []string
	for _, o := range strings.Split(corsRaw, ",") {
		if t := strings.TrimSpace(o); t != "" {
			corsOrigins = append(corsOrigins, t)
		}
	}

	return &Config{
		DatabaseURL:       viper.GetString("DATABASE_URL"),
		RedisURL:          viper.GetString("REDIS_URL"),
		JWTSecret:         viper.GetString("JWT_SECRET"),
		HMACSecret:        viper.GetString("HMAC_SECRET"),
		AppID:             viper.GetString("APP_ID"),
		DevSMSCode:        viper.GetString("DEV_SMS_CODE"),
		PlatformHTTPPort:  viper.GetString("PLATFORM_HTTP_PORT"),
		PitayaWSPort:      viper.GetInt("PITAYA_WS_PORT"),
		PitayaWSURL:       viper.GetString("PITAYA_WS_URL"),
		SnowflakeWorkerID: viper.GetInt64("SNOWFLAKE_WORKER_ID"),
		CORSOrigins:       corsOrigins,
	}, nil
}
