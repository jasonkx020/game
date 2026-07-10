package tools

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/jmoiron/sqlx"
)

func RegisterSuggestAction(reg *Registry, db *sqlx.DB) {
	reg.Register(Definition{
		Name:        "suggest_action",
		Description: "基于公开规则与陪玩提示给出非作弊策略建议",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"game_id": map[string]interface{}{"type": "string"},
				"phase":   map[string]interface{}{"type": "string"},
			},
			"required": []string{"game_id"},
		},
	}, func(ctx context.Context, _ *Context, raw json.RawMessage) (Result, error) {
		var args struct {
			GameID string `json:"game_id"`
			Phase  string `json:"phase"`
		}
		_ = json.Unmarshal(raw, &args)
		if args.GameID == "" {
			args.GameID = "dawugui"
		}
		var tips []string
		err := db.SelectContext(ctx, &tips, `
			SELECT content FROM game_knowledge
			WHERE game_id=$1 AND chunk_title ILIKE '%陪玩%'
			ORDER BY id`, args.GameID)
		if err != nil {
			return Result{}, err
		}
		if len(tips) == 0 {
			tips = []string{"先熟悉基本规则，稳扎稳打即可。"}
		}
		out, _ := json.Marshal(map[string]interface{}{
			"game_id": args.GameID, "phase": args.Phase,
			"tips": strings.Split(tips[0], "；"),
		})
		return Result{Name: "suggest_action", Content: string(out)}, nil
	})
}
