package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/example/auth-service/internal/api"
	"github.com/example/auth-service/internal/authsvc"
	"github.com/example/auth-service/internal/config"
	"github.com/example/auth-service/internal/kyc"
	"github.com/example/auth-service/internal/oauth/wechat"
	"github.com/example/auth-service/internal/otp"
	"github.com/example/auth-service/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	st, err := store.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer st.Close()

	ctx := context.Background()
	if err := st.EnsureAdminSeed(ctx, cfg.AdminDevPhone, "平台管理员"); err != nil {
		log.Printf("admin seed: %v", err)
	}

	var rdb *goredis.Client
	if cfg.RedisURL != "" {
		opt, err := goredis.ParseURL(cfg.RedisURL)
		if err != nil {
			log.Fatalf("redis url: %v", err)
		}
		rdb = goredis.NewClient(opt)
		if err := rdb.Ping(ctx).Err(); err != nil {
			log.Printf("redis ping: %v (nonce/otp redis disabled)", err)
			rdb = nil
		}
	}

	tokens, err := cfg.LoadTokenManager()
	if err != nil {
		log.Fatalf("jwt keys: %v", err)
	}
	otpStore := otp.New(rdb, cfg.DevSMSCode, cfg.DevEmailCode)
	wx := wechat.NewResolver(wechat.Config{
		Mode:      cfg.WeChatMode,
		MPAppID:   cfg.WeChatMPAppID,
		MPSecret:  cfg.WeChatMPSecret,
		WebAppID:  cfg.WeChatWebAppID,
		WebSecret: cfg.WeChatWebSecret,
	})
	verifier := kyc.NewVerifier(cfg.KYCMode, cfg.AliyunAccessKeyID, cfg.AliyunAccessKeySecret, cfg.AliyunCloudauthEndpoint)
	svc := authsvc.New(st, tokens, otpStore, wx, verifier, cfg.RefreshTokenExpire, cfg.RefreshGrace, "")
	srv := api.New(cfg, svc, tokens, rdb)

	httpSrv := &http.Server{
		Addr:              cfg.AuthAPIAddr(),
		Handler:           srv.Engine(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("auth-api listening on %s", httpSrv.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutdownCtx)
}
