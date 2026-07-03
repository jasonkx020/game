package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/example/game/internal/audit"
	"github.com/example/game/internal/config"
	"github.com/example/game/internal/db"
	"github.com/example/game/internal/platform/api"
	"github.com/example/game/internal/platform/catalog"
	"github.com/example/game/internal/platform/club"
	"github.com/example/game/internal/platform/metrics"
	"github.com/example/game/internal/platform/room"
	"github.com/example/game/internal/platform/user"
	"github.com/example/game/internal/platform/wallet"
	iredis "github.com/example/game/internal/redis"
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
	users := user.NewService(sqlDB, cfg.JWTSecret, cfg.DevSMSCode)
	wallets := wallet.NewService(sqlDB)
	rooms := room.NewService(sqlDB)
	clubs := club.NewService(sqlDB)
	catalogSvc := catalog.NewService(sqlDB)
	metricsSvc := metrics.NewService(sqlDB)

	srv := api.New(cfg, users, wallets, rooms, clubs, catalogSvc, metricsSvc, gen, rdb)
	r := gin.Default()
	srv.Register(r)

	addr := ":" + cfg.PlatformHTTPPort
	slog.Info("platform-api listening", "addr", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}
