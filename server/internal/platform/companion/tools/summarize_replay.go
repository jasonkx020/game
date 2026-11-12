package tools

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

func RegisterSummarizeReplay(reg *Registry, db *sqlx.DB) {
	reg.Register(Definition{
		Name:        "summarize_replay",
		Description: "为用户最近一局生成简短复盘摘要（战绩伴侣）",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"game_id": map[string]interface{}{"type": "string"},
			},
		},
	}, func(ctx context.Context, tc *Context, raw json.RawMessage) (Result, error) {
		var args struct {
			GameID string `json:"game_id"`
		}
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &args)
		}
		q := `
			SELECT gr.round_id::text, gr.game_id,
			       (SELECT COUNT(*)::int FROM game_action_log gal WHERE gal.round_id = gr.round_id) AS action_count,
			       ($1 = ANY(gr.winner_user_ids)) AS won
			FROM game_round gr
			JOIN room r ON r.room_id = gr.room_id
			WHERE r.owner_id = $1 AND gr.status = 'ended'`
		argsList := []interface{}{tc.UserID}
		if args.GameID != "" {
			q += ` AND gr.game_id = $2`
			argsList = append(argsList, args.GameID)
		}
		q += ` ORDER BY gr.ended_at DESC NULLS LAST LIMIT 1`

		var row struct {
			RoundID     string `db:"round_id"`
			GameID      string `db:"game_id"`
			ActionCount int    `db:"action_count"`
			Won         bool   `db:"won"`
		}
		err := db.GetContext(ctx, &row, q, argsList...)
		if errors.Is(err, sql.ErrNoRows) || err != nil {
			out, _ := json.Marshal(map[string]string{
				"summary": "还没有可复盘的战绩，先来一局吧！",
			})
			return Result{Name: "summarize_replay", Content: string(out)}, nil
		}
		result := "惜败"
		if row.Won {
			result = "胜利"
		}
		summary := fmt.Sprintf("上一局 %s %s，共 %d 手出牌。%s",
			row.GameID, result, row.ActionCount, encouragement(row.Won))
		out, _ := json.Marshal(map[string]interface{}{
			"round_id": row.RoundID, "game_id": row.GameID,
			"won": row.Won, "action_count": row.ActionCount, "summary": summary,
		})
		return Result{Name: "summarize_replay", Content: string(out)}, nil
	})
}

func encouragement(won bool) string {
	if won {
		return "打得不错，要不要趁热再来一局？"
	}
	return "别灰心，我陪你练练手～"
}
