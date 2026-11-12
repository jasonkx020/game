package roommeta

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var ErrNotFound = errors.New("room not found")

type Meta struct {
	GameID      string          `db:"game_id"`
	PlayerCount int             `db:"player_count"`
	Config      json.RawMessage `db:"config"`
}

func (m *Meta) FillBots() bool {
	if len(m.Config) == 0 {
		return false
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(m.Config, &cfg); err != nil {
		return false
	}
	v, ok := cfg["fill_bots"].(bool)
	return ok && v
}

type Repo struct {
	db *sqlx.DB
}

func NewRepo(db *sqlx.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) Get(ctx context.Context, roomID uuid.UUID) (*Meta, error) {
	var m Meta
	err := r.db.GetContext(ctx, &m,
		`SELECT game_id, player_count, config FROM room WHERE room_id=$1`, roomID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}
