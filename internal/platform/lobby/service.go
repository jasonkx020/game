package lobby

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"time"

	"github.com/jmoiron/sqlx"
)

var ErrInvalidGame = errors.New("invalid game_id")

type BundleInfo struct {
	Version     string `json:"version"`
	URL         string `json:"url"`
	SizeBytes   int64  `json:"size_bytes"`
	SHA256      string `json:"sha256,omitempty"`
	EntryScene  string `json:"entry_scene"`
	MinHostVer  string `json:"min_host_version,omitempty"`
}

type LobbyGameItem struct {
	GameID        string      `json:"game_id"`
	Name          string      `json:"name"`
	IconURL       string      `json:"icon_url"`
	Description   string      `json:"description,omitempty"`
	MinPlayers    int         `json:"min_players"`
	MaxPlayers    int         `json:"max_players"`
	Visible       bool        `json:"visible"`
	Pinned        bool        `json:"pinned"`
	SortOrder     int         `json:"sort_order"`
	LastPlayedAt  *time.Time  `json:"last_played_at,omitempty"`
	Bundle        *BundleInfo `json:"bundle,omitempty"`
}

type GamePrefUpdate struct {
	GameID    string `json:"game_id"`
	Visible   *bool  `json:"visible"`
	Pinned    *bool  `json:"pinned"`
	SortOrder *int   `json:"sort_order"`
}

type catalogRow struct {
	GameID      string         `db:"game_id"`
	Name        string         `db:"name"`
	IconURL     string         `db:"icon_url"`
	Description string         `db:"description"`
	MinPlayers  int            `db:"min_players"`
	MaxPlayers  int            `db:"max_players"`
	CatalogSort int            `db:"catalog_sort"`
	BundleVer   sql.NullString `db:"bundle_version"`
	BundleURL   sql.NullString `db:"bundle_url"`
	BundleSize  sql.NullInt64  `db:"bundle_size_bytes"`
	BundleSHA   sql.NullString `db:"bundle_sha256"`
	EntryScene  sql.NullString `db:"entry_scene"`
	MinHostVer  sql.NullString `db:"min_host_version"`
}

type prefRow struct {
	GameID       string     `db:"game_id"`
	Visible      bool       `db:"visible"`
	Pinned       bool       `db:"pinned"`
	SortOrder    int        `db:"sort_order"`
	LastPlayedAt *time.Time `db:"last_played_at"`
}

type Service struct {
	db *sqlx.DB
}

func NewService(db *sqlx.DB) *Service {
	return &Service{db: db}
}

func (s *Service) ListLobbyGames(ctx context.Context, userID int64) ([]LobbyGameItem, error) {
	var rows []catalogRow
	err := s.db.SelectContext(ctx, &rows, `
		SELECT c.game_id, c.name, c.icon_url, c.description,
		       c.min_players, c.max_players, c.sort_order AS catalog_sort,
		       b.bundle_version, b.bundle_url, b.bundle_size_bytes,
		       b.bundle_sha256, b.entry_scene, b.min_host_version
		FROM game_catalog c
		LEFT JOIN game_client_bundle b ON b.game_id = c.game_id
		WHERE c.enabled = true
		ORDER BY c.sort_order, c.game_id`)
	if err != nil {
		return nil, err
	}

	prefs := map[string]prefRow{}
	var prefRows []prefRow
	_ = s.db.SelectContext(ctx, &prefRows,
		`SELECT game_id, visible, pinned, sort_order, last_played_at FROM user_game_prefs WHERE user_id=$1`, userID)
	for _, p := range prefRows {
		prefs[p.GameID] = p
	}

	items := make([]LobbyGameItem, 0, len(rows))
	for _, r := range rows {
		p, hasPref := prefs[r.GameID]
		visible := true
		pinned := false
		sortOrder := r.CatalogSort
		if hasPref {
			visible = p.Visible
			pinned = p.Pinned
			sortOrder = p.SortOrder
		}
		item := LobbyGameItem{
			GameID:      r.GameID,
			Name:        r.Name,
			IconURL:     r.IconURL,
			Description: r.Description,
			MinPlayers:  r.MinPlayers,
			MaxPlayers:  r.MaxPlayers,
			Visible:     visible,
			Pinned:      pinned,
			SortOrder:   sortOrder,
		}
		if hasPref && p.LastPlayedAt != nil {
			item.LastPlayedAt = p.LastPlayedAt
		}
		if r.BundleURL.Valid && r.BundleURL.String != "" {
			item.Bundle = &BundleInfo{
				Version:    r.BundleVer.String,
				URL:        r.BundleURL.String,
				SizeBytes:  r.BundleSize.Int64,
				SHA256:     r.BundleSHA.String,
				EntryScene: r.EntryScene.String,
				MinHostVer: r.MinHostVer.String,
			}
		}
		items = append(items, item)
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Pinned != items[j].Pinned {
			return items[i].Pinned
		}
		if items[i].SortOrder != items[j].SortOrder {
			return items[i].SortOrder < items[j].SortOrder
		}
		return items[i].GameID < items[j].GameID
	})
	return items, nil
}

func (s *Service) UpdateLobbyGames(ctx context.Context, userID int64, updates []GamePrefUpdate) error {
	for _, u := range updates {
		if u.GameID == "" {
			continue
		}
		var exists bool
		if err := s.db.GetContext(ctx, &exists,
			`SELECT EXISTS(SELECT 1 FROM game_catalog WHERE game_id=$1 AND enabled)`, u.GameID); err != nil {
			return err
		}
		if !exists {
			return ErrInvalidGame
		}

		cur := prefRow{GameID: u.GameID, Visible: true, Pinned: false, SortOrder: 0}
		_ = s.db.GetContext(ctx, &cur,
			`SELECT game_id, visible, pinned, sort_order FROM user_game_prefs WHERE user_id=$1 AND game_id=$2`,
			userID, u.GameID)

		if u.Visible != nil {
			cur.Visible = *u.Visible
		}
		if u.Pinned != nil {
			cur.Pinned = *u.Pinned
		}
		if u.SortOrder != nil {
			cur.SortOrder = *u.SortOrder
		}

		_, err := s.db.ExecContext(ctx, `
			INSERT INTO user_game_prefs (user_id, game_id, visible, pinned, sort_order, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (user_id, game_id) DO UPDATE SET
				visible = EXCLUDED.visible,
				pinned = EXCLUDED.pinned,
				sort_order = EXCLUDED.sort_order,
				updated_at = EXCLUDED.updated_at`,
			userID, u.GameID, cur.Visible, cur.Pinned, cur.SortOrder, time.Now())
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) TouchLastPlayed(ctx context.Context, userID int64, gameID string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO user_game_prefs (user_id, game_id, last_played_at, updated_at)
		VALUES ($1, $2, $3, $3)
		ON CONFLICT (user_id, game_id) DO UPDATE SET
			last_played_at = $3, updated_at = $3`,
		userID, gameID, time.Now())
	return err
}
