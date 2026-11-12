package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/example/game/internal/audit"
	"github.com/example/game/internal/config"
	"github.com/example/game/internal/platform/catalog"
	"github.com/example/game/internal/platform/lobby"
	"github.com/example/game/internal/platform/room"
)

type RoomDeps struct {
	Rooms   *room.Service
	Catalog *catalog.Service
	Lobby   *lobby.Service
	Cfg     *config.Config
	Audit   *audit.Generator
}

func RegisterCreateRoom(reg *Registry, deps RoomDeps) {
	reg.Register(Definition{
		Name:        "create_room",
		Description: "为用户创建房卡场房间",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"game_id":      map[string]interface{}{"type": "string"},
				"player_count": map[string]interface{}{"type": "integer"},
			},
			"required": []string{"game_id"},
		},
	}, func(ctx context.Context, tc *Context, raw json.RawMessage) (Result, error) {
		var args struct {
			GameID      string `json:"game_id"`
			PlayerCount int    `json:"player_count"`
			FillBots    bool   `json:"fill_bots"`
		}
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &args)
		}
		if args.GameID == "" {
			args.GameID = "dawugui"
		}
		if args.PlayerCount == 0 {
			args.PlayerCount = 4
		}
		if err := deps.Catalog.ValidatePlayerCount(ctx, args.GameID, args.PlayerCount); err != nil {
			return Result{Name: "create_room", Content: fmt.Sprintf(`{"error":%q}`, err.Error())}, nil
		}
		cost, err := deps.Catalog.RoomCardCost(ctx, args.GameID, args.PlayerCount)
		if err != nil {
			return Result{}, err
		}
		auditSN := deps.Audit.Next()
		r, err := deps.Rooms.Create(ctx, room.CreateParams{
			OwnerID: tc.UserID, GameID: args.GameID, RoomMode: "room_card",
			PlayerCount: args.PlayerCount, Config: map[string]interface{}{
				"base_score": 1, "fill_bots": args.FillBots,
			},
			WSURL: deps.Cfg.PitayaWSURL, RoomCardCost: cost, AuditSN: auditSN,
		})
		if err != nil {
			return Result{Name: "create_room", Content: fmt.Sprintf(`{"error":%q}`, err.Error())}, nil
		}
		_ = deps.Lobby.TouchLastPlayed(ctx, tc.UserID, args.GameID)
		out, _ := json.Marshal(map[string]interface{}{
			"room_id": r.RoomID.String(), "ws_url": r.WSURL, "game_id": r.GameID, "cost": cost,
		})
		return Result{Name: "create_room", Content: string(out)}, nil
	})
}
