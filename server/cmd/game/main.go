package main

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/topfreegames/pitaya/v2"
	"github.com/topfreegames/pitaya/v2/acceptor"
	"github.com/topfreegames/pitaya/v2/component"
	pitayaconfig "github.com/topfreegames/pitaya/v2/config"
	"github.com/topfreegames/pitaya/v2/serialize/protobuf"
	"github.com/topfreegames/pitaya/v2/session"

	"github.com/example/game/internal/pitaya/commit"
	"github.com/example/game/internal/pitaya/handlers/connector"
	"github.com/example/game/internal/pitaya/handlers/dawugui"
	"github.com/example/game/internal/pitaya/handlers/liuzichong"
	roomh "github.com/example/game/internal/pitaya/handlers/room"
	"github.com/example/game/internal/pitaya/opskick"
	"github.com/example/game/internal/pitaya/runtime"
	"github.com/example/game/internal/platform/actionlog"
	"github.com/example/game/pkg/audit"
	"github.com/example/game/pkg/config"
	"github.com/example/game/pkg/db"
	"github.com/example/game/pkg/platform/player"
	"github.com/example/game/pkg/platform/risk"
	"github.com/example/game/pkg/platform/settlement"
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
	logRepo := actionlog.NewRepo(sqlDB)
	players := player.NewService(sqlDB)
	wallets := wallet.NewService(sqlDB)
	settler := settlement.New(wallets)
	riskSvc := risk.NewService(sqlDB, risk.Config{
		GPSHardBlock: cfg.RiskGPSHardBlock,
		AutoFreeze:   cfg.RiskAutoFreeze,
	})
	store := runtime.NewStore()
	committer := &commit.Committer{Log: logRepo, Audit: gen}

	conf := pitayaconfig.NewDefaultPitayaConfig()
	// Client Pitaya codec historically lacked inflate; keep off unless both sides support zlib.
	conf.Handler.Messages.Compression = false
	builder := pitaya.NewDefaultBuilder(true, "game", pitaya.Standalone, map[string]string{}, *conf)
	builder.Serializer = protobuf.NewSerializer()
	builder.AddAcceptor(acceptor.NewWSAcceptor(cfg.PitayaWSAddr()))
	app := builder.Build()
	pitaya.DefaultApp = app
	session.DefaultSessionPool = builder.SessionPool

	// Client routes use lowercase methods (game.connector.entry); Pitaya defaults to Entry.
	nameOpts := []component.Option{component.WithNameFunc(strings.ToLower)}
	pitaya.Register(connector.New(cfg, players, riskSvc), append([]component.Option{component.WithName("connector")}, nameOpts...)...)
	pitaya.Register(roomh.New(store, committer, gen, logRepo, sqlDB, wallets, settler, riskSvc), append([]component.Option{component.WithName("room")}, nameOpts...)...)
	pitaya.Register(dawugui.New(store, committer, gen, logRepo, wallets, settler, riskSvc), append([]component.Option{component.WithName("dawugui")}, nameOpts...)...)
	pitaya.Register(liuzichong.New(store, committer, gen, logRepo, wallets, settler, riskSvc), append([]component.Option{component.WithName("liuzichong")}, nameOpts...)...)

	kickCtx, kickCancel := context.WithCancel(context.Background())
	defer kickCancel()
	opskick.StartSubscribe(kickCtx, rdb)

	slog.Info("game server listening", "addr", cfg.PitayaWSAddr())
	pitaya.Start()
}
