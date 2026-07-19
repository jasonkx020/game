package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

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
	// Client Pitaya codec historically lacked inflate; keep off unless both sides support zlib.
	conf.Handler.Messages.Compression = false
	builder := pitaya.NewDefaultBuilder(true, "game", pitaya.Standalone, map[string]string{}, *conf)
	builder.Serializer = protobuf.NewSerializer()
	builder.AddAcceptor(acceptor.NewWSAcceptor(fmt.Sprintf(":%d", cfg.PitayaWSPort)))
	app := builder.Build()
	pitaya.DefaultApp = app
	session.DefaultSessionPool = builder.SessionPool

	// Client routes use lowercase methods (game.connector.entry); Pitaya defaults to Entry.
	nameOpts := []component.Option{component.WithNameFunc(strings.ToLower)}
	pitaya.Register(connector.New(cfg), append([]component.Option{component.WithName("connector")}, nameOpts...)...)
	pitaya.Register(roomh.New(store, committer, gen, logRepo, sqlDB), append([]component.Option{component.WithName("room")}, nameOpts...)...)
	pitaya.Register(dawugui.New(store, committer, gen, logRepo), append([]component.Option{component.WithName("dawugui")}, nameOpts...)...)
	pitaya.Register(liuzichong.New(store, committer, gen, logRepo), append([]component.Option{component.WithName("liuzichong")}, nameOpts...)...)

	slog.Info("game server listening", "port", cfg.PitayaWSPort)
	pitaya.Start()
}
