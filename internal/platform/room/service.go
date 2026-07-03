package room

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var ErrNotFound = errors.New("room not found")
var ErrIdempotencyConflict = errors.New("idempotency conflict")
var ErrInsufficientRoomCard = errors.New("insufficient room card")
var ErrInsufficientClubPool = errors.New("insufficient club pool")

type Room struct {
	RoomID      uuid.UUID       `db:"room_id" json:"room_id"`
	OwnerID     int64           `db:"owner_id" json:"owner_id"`
	ClubID      *int64          `db:"club_id" json:"club_id,omitempty"`
	GameID      string          `db:"game_id" json:"game_id"`
	RoomMode    string          `db:"room_mode" json:"room_mode"`
	Status      string          `db:"status" json:"status"`
	PlayerCount int             `db:"player_count" json:"player_count"`
	Config      json.RawMessage `db:"config" json:"config"`
	WSURL       string          `db:"ws_url" json:"ws_url"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
}

type CreateParams struct {
	OwnerID        int64
	ClubID         *int64
	GameID         string
	RoomMode       string
	PlayerCount    int
	Config         map[string]interface{}
	WSURL          string
	IdempotencyKey string
	RoomCardCost   int64
	AuditSN        uint64
}

type Service struct {
	db *sqlx.DB
}

func NewService(db *sqlx.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Create(ctx context.Context, p CreateParams) (*Room, error) {
	if p.IdempotencyKey != "" {
		var existing Room
		err := s.db.GetContext(ctx, &existing,
			`SELECT room_id, owner_id, club_id, game_id, room_mode, status, player_count, config, ws_url, created_at
			 FROM room WHERE idempotency_key=$1`, p.IdempotencyKey)
		if err == nil {
			return &existing, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}

	cfgBytes, _ := json.Marshal(p.Config)
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	useClubPool := p.ClubID != nil && *p.ClubID > 0
	var pendingPersonalDeduct bool
	var personalBalBefore int64
	if useClubPool {
		var poolBal int64
		if err := tx.GetContext(ctx, &poolBal,
			`SELECT balance FROM club_room_card_pool WHERE club_id=$1 FOR UPDATE`, *p.ClubID); err != nil {
			return nil, err
		}
		if poolBal < p.RoomCardCost {
			return nil, ErrInsufficientClubPool
		}
		newPool := poolBal - p.RoomCardCost
		if _, err := tx.ExecContext(ctx,
			`UPDATE club_room_card_pool SET balance=$1, updated_at=now() WHERE club_id=$2`, newPool, *p.ClubID); err != nil {
			return nil, err
		}
	} else {
		var bal int64
		if err := tx.GetContext(ctx, &bal,
			`SELECT balance FROM wallet_room_card WHERE user_id=$1 FOR UPDATE`, p.OwnerID); err != nil {
			return nil, err
		}
		if bal < p.RoomCardCost {
			return nil, ErrInsufficientRoomCard
		}
		pendingPersonalDeduct = true
		personalBalBefore = bal
	}

	var room Room
	idKey := sql.NullString{}
	if p.IdempotencyKey != "" {
		idKey = sql.NullString{String: p.IdempotencyKey, Valid: true}
	}
	var clubID sql.NullInt64
	if useClubPool {
		clubID = sql.NullInt64{Int64: *p.ClubID, Valid: true}
	}
	err = tx.QueryRowxContext(ctx,
		`INSERT INTO room (owner_id, club_id, game_id, room_mode, player_count, config, ws_url, idempotency_key)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		 RETURNING room_id, owner_id, club_id, game_id, room_mode, status, player_count, config, ws_url, created_at`,
		p.OwnerID, clubID, p.GameID, p.RoomMode, p.PlayerCount, cfgBytes, p.WSURL, idKey).StructScan(&room)
	if err != nil {
		return nil, err
	}

	if pendingPersonalDeduct {
		newBal := personalBalBefore - p.RoomCardCost
		if _, err := tx.ExecContext(ctx,
			`UPDATE wallet_room_card SET balance=$1, updated_at=now() WHERE user_id=$2`, newBal, p.OwnerID); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO wallet_ledger (user_id, wallet_type, delta, balance_after, reason, ref_id, audit_sn)
			 VALUES ($1,'room_card',$2,$3,'create_room',$4,$5)`,
			p.OwnerID, -p.RoomCardCost, newBal, room.RoomID, p.AuditSN); err != nil {
			return nil, err
		}
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO room_event_log (room_id, room_seq, event_type, user_id, audit_sn, payload)
		 VALUES ($1, 1, 'ROOM_CREATED', $2, $3, '{}')`,
		room.RoomID, p.OwnerID, p.AuditSN+1); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &room, nil
}

func (s *Service) Get(ctx context.Context, roomID uuid.UUID) (*Room, error) {
	var r Room
	if err := s.db.GetContext(ctx, &r,
		`SELECT room_id, owner_id, club_id, game_id, room_mode, status, player_count, config, ws_url, created_at
		 FROM room WHERE room_id=$1`, roomID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &r, nil
}

func (s *Service) JoinAllowed(ctx context.Context, roomID uuid.UUID, userID int64) (string, error) {
	r, err := s.Get(ctx, roomID)
	if err != nil {
		return "", err
	}
	if r.Status != "waiting" && r.Status != "playing" {
		return "", errors.New("room not joinable")
	}
	_ = userID
	return r.WSURL, nil
}
