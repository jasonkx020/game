package tools

import (
	"context"
	"encoding/json"

	"github.com/example/game/internal/platform/lobby"
)

func RegisterListGames(reg *Registry, lobbySvc *lobby.Service) {
	reg.Register(Definition{
		Name:        "list_games",
		Description: "列出大厅可用游戏及用户偏好",
		Parameters: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	}, func(ctx context.Context, tc *Context, _ json.RawMessage) (Result, error) {
		games, err := lobbySvc.ListLobbyGames(ctx, tc.UserID)
		if err != nil {
			return Result{}, err
		}
		b, _ := json.Marshal(games)
		return Result{Name: "list_games", Content: string(b)}, nil
	})
}
