package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/example/admin-service/internal/api"
	"github.com/example/admin-service/internal/authclient"
	"github.com/example/game/pkg/audit"
	"github.com/example/game/pkg/config"
	"github.com/example/game/pkg/db"
	"github.com/example/game/pkg/platform/adminuser"
	"github.com/example/game/pkg/platform/catalog"
	"github.com/example/game/pkg/platform/club"
	"github.com/example/game/pkg/platform/metrics"
	"github.com/example/game/pkg/platform/moderation"
	"github.com/example/game/pkg/platform/player"
	"github.com/example/game/pkg/platform/wallet"
	iredis "github.com/example/game/pkg/redis"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	sqlDB, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		slog.Error("db", "err", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	rdb, err := iredis.Connect(cfg.RedisURL)
	if err != nil {
		slog.Error("redis", "err", err)
		os.Exit(1)
	}
	defer rdb.Close()

	gen := audit.NewGenerator(cfg.SnowflakeWorkerID)
	admins := adminuser.NewService(sqlDB)
	players := player.NewService(sqlDB)
	wallets := wallet.NewService(sqlDB)
	clubs := club.NewService(sqlDB)
	catalogSvc := catalog.NewService(sqlDB)
	metricsSvc := metrics.NewService(sqlDB)
	modSvc := moderation.NewService(sqlDB)
	authCLI := authclient.New(cfg.AuthBaseURL, cfg.AppID, cfg.HMACSecret)

	srv := api.New(cfg, admins, players, wallets, clubs, catalogSvc, metricsSvc, modSvc, authCLI, gen, rdb, sqlDB)
	r := gin.Default()
	srv.Register(r)

	addr := ":" + cfg.AdminHTTPPort
	slog.Info("admin-api listening", "addr", addr, "auth", cfg.AuthBaseURL)
	if err := http.ListenAndServe(addr, r); err != nil {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}
