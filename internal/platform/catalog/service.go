package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/jmoiron/sqlx"
)

var ErrNotFound = errors.New("game not found")

type GameSummary struct {
	GameID     string `db:"game_id" json:"game_id"`
	Name       string `db:"name" json:"name"`
	MinPlayers int    `db:"min_players" json:"min_players"`
	MaxPlayers int    `db:"max_players" json:"max_players"`
	Enabled    bool   `db:"enabled" json:"enabled"`
}

type GameConfig struct {
	GameID       string                 `json:"game_id"`
	RoomCardCost map[string]int64       `json:"room_card_cost"`
	RegisterGift map[string]int64       `json:"register_gift"`
	CoinName     string                 `json:"coin_name"`
	Raw          map[string]interface{} `json:"-"`
}

type Service struct {
	db *sqlx.DB
}

func NewService(db *sqlx.DB) *Service {
	return &Service{db: db}
}

func (s *Service) ListGames(ctx context.Context) ([]GameSummary, error) {
	var games []GameSummary
	err := s.db.SelectContext(ctx, &games,
		`SELECT game_id, name, min_players, max_players, enabled FROM game_catalog WHERE enabled = true ORDER BY game_id`)
	return games, err
}

func (s *Service) GetGameConfig(ctx context.Context, gameID string) (*GameConfig, error) {
	var raw json.RawMessage
	err := s.db.GetContext(ctx, &raw,
		`SELECT config FROM game_ops_config WHERE game_id=$1`, gameID)
	if err != nil {
		return nil, ErrNotFound
	}
	var cfg GameConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	cfg.GameID = gameID
	_ = json.Unmarshal(raw, &cfg.Raw)
	return &cfg, nil
}

func (s *Service) RoomCardCost(ctx context.Context, gameID string, playerCount int) (int64, error) {
	cfg, err := s.GetGameConfig(ctx, gameID)
	if err != nil {
		return defaultRoomCardCost(playerCount), nil
	}
	key := strconv.Itoa(playerCount)
	if cost, ok := cfg.RoomCardCost[key]; ok {
		return cost, nil
	}
	return defaultRoomCardCost(playerCount), nil
}

func defaultRoomCardCost(playerCount int) int64 {
	switch playerCount {
	case 3, 4:
		return 2
	case 5:
		return 3
	default:
		return 2
	}
}

func (s *Service) ValidatePlayerCount(ctx context.Context, gameID string, playerCount int) error {
	var g GameSummary
	err := s.db.GetContext(ctx, &g,
		`SELECT game_id, name, min_players, max_players, enabled FROM game_catalog WHERE game_id=$1 AND enabled`, gameID)
	if err != nil {
		return ErrNotFound
	}
	if playerCount < g.MinPlayers || playerCount > g.MaxPlayers {
		return fmt.Errorf("player_count must be between %d and %d", g.MinPlayers, g.MaxPlayers)
	}
	return nil
}
