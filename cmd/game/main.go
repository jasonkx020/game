package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/topfreegames/pitaya/v2"
	"github.com/topfreegames/pitaya/v2/acceptor"
	"github.com/topfreegames/pitaya/v2/component"
	pitayaconfig "github.com/topfreegames/pitaya/v2/config"
	"github.com/topfreegames/pitaya/v2/serialize/protobuf"
	"github.com/topfreegames/pitaya/v2/session"

	"github.com/example/game/internal/audit"
	"github.com/example/game/internal/config"
	"github.com/example/game/internal/db"
	"github.com/example/game/internal/pitaya/commit"
	"github.com/example/game/internal/pitaya/handlers/connector"
	"github.com/example/game/internal/pitaya/handlers/dawugui"
	"github.com/example/game/internal/pitaya/handlers/liuzichong"
	roomh "github.com/example/game/internal/pitaya/handlers/room"
	"github.com/example/game/internal/pitaya/runtime"
	"github.com/example/game/internal/platform/actionlog"
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
	_ = rdb

	gen := audit.NewGenerator(cfg.SnowflakeWorkerID)
	logRepo := actionlog.NewRepo(sqlDB)
	store := runtime.NewStore()
	committer := &commit.Committer{Log: logRepo, Audit: gen}

	conf := pitayaconfig.NewDefaultPitayaConfig()
	builder := pitaya.NewDefaultBuilder(true, "game", pitaya.Standalone, map[string]string{}, *conf)
	builder.Serializer = protobuf.NewSerializer()
	builder.AddAcceptor(acceptor.NewWSAcceptor(fmt.Sprintf(":%d", cfg.PitayaWSPort)))
	app := builder.Build()
	pitaya.DefaultApp = app
	session.DefaultSessionPool = builder.SessionPool

	pitaya.Register(connector.New(cfg), component.WithName("connector"))
	pitaya.Register(roomh.New(store, committer, gen, logRepo, sqlDB), component.WithName("room"))
	pitaya.Register(dawugui.New(store, committer, gen, logRepo), component.WithName("dawugui"))
	pitaya.Register(liuzichong.New(store, committer, gen, logRepo), component.WithName("liuzichong"))

	slog.Info("game server listening", "port", cfg.PitayaWSPort)
	pitaya.Start()
}
