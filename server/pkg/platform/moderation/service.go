package moderation

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jmoiron/sqlx"
)

const (
	TargetPlayer = "player"
	TargetAdmin  = "admin"

	ActionBan            = "ban"
	ActionUnban          = "unban"
	ActionKick           = "kick"
	ActionDisable        = "disable"
	ActionEnable         = "enable"
	ActionSetRole        = "set_role"
	ActionCreate         = "create"
	ActionUpdateNickname = "update_nickname"
	ActionForceLogout    = "force_logout"
)

type Service struct {
	db *sqlx.DB
}

func NewService(db *sqlx.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Record(ctx context.Context, actorAdminID int64, targetType string, targetID int64, action, reason string, meta map[string]interface{}) error {
	if meta == nil {
		meta = map[string]interface{}{}
	}
	b, _ := json.Marshal(meta)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO moderation_actions (actor_admin_id, target_type, target_id, action, reason, meta, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		actorAdminID, targetType, targetID, action, reason, b, time.Now())
	return err
}

type ActionRow struct {
	ID           int64           `json:"id" db:"id"`
	ActorAdminID int64           `json:"actor_admin_id" db:"actor_admin_id"`
	TargetType   string          `json:"target_type" db:"target_type"`
	TargetID     int64           `json:"target_id" db:"target_id"`
	Action       string          `json:"action" db:"action"`
	Reason       string          `json:"reason" db:"reason"`
	Meta         json.RawMessage `json:"meta" db:"meta"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
}

// ListByTarget returns recent moderation actions for a target (newest first).
func (s *Service) ListByTarget(ctx context.Context, targetType string, targetID int64, limit int) ([]ActionRow, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	var rows []ActionRow
	err := s.db.SelectContext(ctx, &rows, `
		SELECT id, actor_admin_id, target_type, target_id, action, reason, meta, created_at
		FROM moderation_actions
		WHERE target_type=$1 AND target_id=$2
		ORDER BY created_at DESC
		LIMIT $3`, targetType, targetID, limit)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []ActionRow{}
	}
	return rows, nil
}
