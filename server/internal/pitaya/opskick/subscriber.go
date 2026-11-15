package opskick

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"

	pitaya "github.com/topfreegames/pitaya/v2"
	goredis "github.com/redis/go-redis/v9"

	pkgkick "github.com/example/game/pkg/opskick"
)

// StartSubscribe listens for kick requests and disconnects Pitaya sessions by player UID.
func StartSubscribe(ctx context.Context, rdb *goredis.Client) {
	if rdb == nil {
		return
	}
	go func() {
		sub := rdb.Subscribe(ctx, pkgkick.Channel)
		defer sub.Close()
		ch := sub.Channel()
		slog.Info("opskick subscribed", "channel", pkgkick.Channel)
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				var p pkgkick.Payload
				if err := json.Unmarshal([]byte(msg.Payload), &p); err != nil || p.PlayerID <= 0 {
					slog.Warn("opskick bad payload", "raw", msg.Payload, "err", err)
					continue
				}
				kickPlayer(p.PlayerID)
			}
		}
	}()
}

func kickPlayer(playerID int64) {
	app := pitaya.DefaultApp
	if app == nil {
		slog.Warn("opskick: pitaya app nil", "player_id", playerID)
		return
	}
	uid := strconv.FormatInt(playerID, 10)
	if _, err := app.SendKickToUsers([]string{uid}, "game"); err != nil {
		slog.Info("opskick result", "player_id", playerID, "err", err)
		return
	}
	slog.Info("opskick kicked", "player_id", playerID)
}
