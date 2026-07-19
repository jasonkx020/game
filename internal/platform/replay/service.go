package replay

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type MatchSummary struct {
	RoundID         string     `json:"round_id"`
	RoomID          string     `json:"room_id"`
	RoundNo         int        `json:"round_no"`
	GameID          string     `json:"game_id"`
	Status          string     `json:"status"`
	StartedAt       time.Time  `json:"started_at"`
	EndedAt         *time.Time `json:"ended_at,omitempty"`
	MyRuleScore     int        `json:"my_rule_score"`
	MyCoinDelta     int64      `json:"my_coin_delta"`
	IsWinner        bool       `json:"is_winner"`
	ReplayAvailable bool       `json:"replay_available"`
}

type matchDBRow struct {
	RoundID     uuid.UUID  `db:"round_id"`
	RoomID      uuid.UUID  `db:"room_id"`
	RoundNo     int        `db:"round_no"`
	GameID      string     `db:"game_id"`
	Status      string     `db:"status"`
	StartedAt   time.Time  `db:"started_at"`
	EndedAt     *time.Time `db:"ended_at"`
	IsWinner    bool       `db:"is_winner"`
	ActionCount int        `db:"action_count"`
}

type Service struct {
	db *sqlx.DB
}

func NewService(db *sqlx.DB) *Service {
	return &Service{db: db}
}

func (s *Service) ListMyMatches(ctx context.Context, playerID int64, gameID string, page, pageSize int) ([]MatchSummary, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 50 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize

	where := `gr.status = 'ended' AND EXISTS (
		SELECT 1 FROM game_action_log gal
		WHERE gal.round_id = gr.round_id AND gal.actor_user_id = $1
	)`
	args := []interface{}{playerID}
	argN := 2
	if gameID != "" {
		where += fmt.Sprintf(` AND gr.game_id = $%d`, argN)
		args = append(args, gameID)
		argN++
	}

	var total int
	countSQL := `SELECT COUNT(*) FROM game_round gr WHERE ` + where
	if err := s.db.GetContext(ctx, &total, countSQL, args...); err != nil {
		return nil, 0, err
	}

	listSQL := fmt.Sprintf(`
		SELECT gr.round_id, gr.room_id, gr.round_no, gr.game_id, gr.status,
		       gr.started_at, gr.ended_at,
		       ($1 = ANY(COALESCE(gr.winner_user_ids, '{}'))) AS is_winner,
		       (SELECT COUNT(*)::int FROM game_action_log gal WHERE gal.round_id = gr.round_id) AS action_count
		FROM game_round gr
		WHERE %s
		ORDER BY gr.ended_at DESC NULLS LAST, gr.started_at DESC
		LIMIT $%d OFFSET $%d`, where, argN, argN+1)
	args = append(args, pageSize, offset)

	var rows []matchDBRow
	if err := s.db.SelectContext(ctx, &rows, listSQL, args...); err != nil {
		return nil, 0, err
	}

	out := make([]MatchSummary, len(rows))
	for i, r := range rows {
		out[i] = MatchSummary{
			RoundID:         r.RoundID.String(),
			RoomID:          r.RoomID.String(),
			RoundNo:         r.RoundNo,
			GameID:          r.GameID,
			Status:          r.Status,
			StartedAt:       r.StartedAt,
			EndedAt:         r.EndedAt,
			MyRuleScore:     0,
			MyCoinDelta:     0,
			IsWinner:        r.IsWinner,
			ReplayAvailable: r.ActionCount > 0,
		}
	}
	return out, total, nil
}
