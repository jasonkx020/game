package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

func RegisterExplainRules(reg *Registry, db *sqlx.DB) {
	reg.Register(Definition{
		Name:        "explain_rules",
		Description: "讲解指定游戏规则与陪玩提示",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"game_id": map[string]interface{}{"type": "string"},
				"query":   map[string]interface{}{"type": "string", "description": "关键词检索 RAG"},
			},
			"required": []string{"game_id"},
		},
	}, func(ctx context.Context, _ *Context, raw json.RawMessage) (Result, error) {
		var args struct {
			GameID string `json:"game_id"`
			Query  string `json:"query"`
		}
		_ = json.Unmarshal(raw, &args)
		if args.GameID == "" {
			args.GameID = "dawugui"
		}
		var chunks []struct {
			Title   string `db:"chunk_title"`
			Content string `db:"content"`
		}
		err := db.SelectContext(ctx, &chunks,
			`SELECT chunk_title, content FROM game_knowledge
			 WHERE game_id=$1 AND ($2 = '' OR content ILIKE '%' || $2 || '%' OR chunk_title ILIKE '%' || $2 || '%')
			 ORDER BY id`, args.GameID, args.Query)
		if err != nil {
			return Result{}, err
		}
		if len(chunks) == 0 {
			return Result{Name: "explain_rules", Content: fmt.Sprintf(`{"game_id":%q,"summary":"暂无规则资料"}`, args.GameID)}, nil
		}
		var sb strings.Builder
		for _, c := range chunks {
			sb.WriteString(c.Title)
			sb.WriteString("：")
			sb.WriteString(c.Content)
			sb.WriteString("\n")
		}
		out, _ := json.Marshal(map[string]string{"game_id": args.GameID, "summary": sb.String()})
		return Result{Name: "explain_rules", Content: string(out)}, nil
	})
}

func RegisterRecommend(reg *Registry, db *sqlx.DB) {
	reg.Register(Definition{
		Name:        "recommend_games",
		Description: "根据最近游玩推荐游戏",
		Parameters: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	}, func(ctx context.Context, tc *Context, _ json.RawMessage) (Result, error) {
		type row struct {
			GameID string `db:"game_id" json:"game_id"`
			Name   string `db:"name" json:"name"`
		}
		var rows []row
		err := db.SelectContext(ctx, &rows, `
			SELECT c.game_id, c.name FROM game_catalog c
			LEFT JOIN user_game_prefs p ON p.game_id = c.game_id AND p.user_id = $1
			WHERE c.enabled = true
			ORDER BY p.last_played_at DESC NULLS LAST, c.sort_order, c.game_id
			LIMIT 5`, tc.UserID)
		if err != nil {
			return Result{}, err
		}
		b, _ := json.Marshal(rows)
		return Result{Name: "recommend_games", Content: string(b)}, nil
	})
}
