package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/example/game/internal/platform/room"
)

func RegisterGetRoomStatus(reg *Registry, rooms *room.Service) {
	reg.Register(Definition{
		Name:        "get_room_status",
		Description: "查询用户当前或指定房间状态",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"room_id": map[string]interface{}{"type": "string"},
			},
		},
	}, func(ctx context.Context, tc *Context, raw json.RawMessage) (Result, error) {
		var args struct {
			RoomID string `json:"room_id"`
		}
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &args)
		}
		if args.RoomID == "" {
			out, _ := json.Marshal(map[string]string{"status": "no_room", "hint": "用户尚未开房"})
			return Result{Name: "get_room_status", Content: string(out)}, nil
		}
		rid, err := uuid.Parse(args.RoomID)
		if err != nil {
			return Result{Name: "get_room_status", Content: fmt.Sprintf(`{"error":%q}`, err.Error())}, nil
		}
		r, err := rooms.Get(ctx, rid)
		if err != nil {
			return Result{Name: "get_room_status", Content: fmt.Sprintf(`{"error":%q}`, err.Error())}, nil
		}
		if r.OwnerID != tc.UserID {
			return Result{Name: "get_room_status", Content: `{"error":"not your room"}`}, nil
		}
		out, _ := json.Marshal(map[string]interface{}{
			"room_id": r.RoomID.String(), "game_id": r.GameID, "status": r.Status,
			"player_count": r.PlayerCount, "ws_url": r.WSURL,
		})
		return Result{Name: "get_room_status", Content: string(out)}, nil
	})
}
