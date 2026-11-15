package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/example/game/pkg/audit"
	"github.com/example/game/pkg/config"
	"github.com/example/game/pkg/db"
	"github.com/example/game/internal/platform/api"
	"github.com/example/game/pkg/platform/catalog"
	"github.com/example/game/pkg/platform/club"
	"github.com/example/game/internal/platform/companion"
	"github.com/example/game/internal/platform/companion/llm"
	"github.com/example/game/internal/platform/lobby"
	"github.com/example/game/pkg/platform/player"
	"github.com/example/game/internal/platform/replay"
	"github.com/example/game/internal/platform/room"
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
	players := player.NewService(sqlDB, cfg.JWTSecret, cfg.DevSMSCode)
	wallets := wallet.NewService(sqlDB)
	rooms := room.NewService(sqlDB)
	clubs := club.NewService(sqlDB)
	catalogSvc := catalog.NewService(sqlDB)
	lobbySvc := lobby.NewService(sqlDB)
	llmClient := llm.NewClient(llm.Config{
		BaseURL: cfg.LLMBaseURL, APIKey: cfg.LLMAPIKey, Model: cfg.LLMModel, TimeoutSec: cfg.LLMTimeoutSec,
	})
	toolReg := companion.NewToolRegistry(sqlDB, lobbySvc, rooms, catalogSvc, cfg, gen)
	companionSvc := companion.NewService(sqlDB, llmClient, toolReg, gen)
	replaySvc := replay.NewService(sqlDB)

	srv := api.New(cfg, players, wallets, rooms, clubs, catalogSvc, lobbySvc, companionSvc, replaySvc, gen, rdb)
	r := gin.Default()
	srv.Register(r)

	addr := ":" + cfg.GameAPIHTTPPort
	slog.Info("game-api listening", "addr", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}
