package actionlog

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Entry struct {
	RoomID        uuid.UUID
	RoundID       uuid.UUID
	ActionSeq     int
	AuditSN       uint64
	EventType     string
	ActorUserID   *int64
	Seat          *int16
	Payload       []byte
	PushRoute     string
	C2SRoute      string
	C2SRequestID  *uuid.UUID
}

type Round struct {
	RoundID        uuid.UUID
	RoomID         uuid.UUID
	RoundNo        int
	GameID         string
	Status         string
	ConfigSnapshot []byte
}

type Repo struct {
	db *sqlx.DB
}

func NewRepo(db *sqlx.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) Insert(ctx context.Context, e Entry) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO game_action_log
		 (room_id, round_id, action_seq, audit_sn, event_type, actor_user_id, seat, payload, push_route, c2s_route, c2s_request_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		e.RoomID, e.RoundID, e.ActionSeq, e.AuditSN, e.EventType, e.ActorUserID, e.Seat,
		e.Payload, e.PushRoute, nullStr(e.C2SRoute), e.C2SRequestID)
	return err
}

func (r *Repo) CreateRound(ctx context.Context, roomID uuid.UUID, roundNo int, gameID string, config []byte) (*Round, error) {
	var rd Round
	err := r.db.QueryRowxContext(ctx,
		`INSERT INTO game_round (room_id, round_no, game_id, status, config_snapshot)
		 VALUES ($1,$2,$3,'playing',$4)
		 RETURNING round_id, room_id, round_no, game_id, status, config_snapshot`,
		roomID, roundNo, gameID, config).StructScan(&rd)
	return &rd, err
}

func (r *Repo) EndRound(ctx context.Context, roundID uuid.UUID, status string, winners []int64, settlementAuditSN uint64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE game_round SET status=$1, ended_at=now(), winner_user_ids=$2, settlement_audit_sn=$3 WHERE round_id=$4`,
		status, winners, settlementAuditSN, roundID)
	return err
}

func (r *Repo) InsertRoomEvent(ctx context.Context, roomID uuid.UUID, roomSeq int, eventType string, userID int64, auditSN uint64, payload []byte) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO room_event_log (room_id, room_seq, event_type, user_id, audit_sn, payload)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		roomID, roomSeq, eventType, userID, auditSN, payload)
	return err
}

func (r *Repo) InsertSettlement(ctx context.Context, roomID, roundID uuid.UUID, gameID string, auditSN uint64, payload []byte) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO settlement_record (room_id, round_id, game_id, audit_sn, payload) VALUES ($1,$2,$3,$4,$5)`,
		roomID, roundID, gameID, auditSN, payload)
	return err
}

func (r *Repo) ListSince(ctx context.Context, roundID uuid.UUID, sinceSeq int) ([]Entry, error) {
	var rows []struct {
		RoomID    uuid.UUID `db:"room_id"`
		RoundID   uuid.UUID `db:"round_id"`
		ActionSeq int       `db:"action_seq"`
		AuditSN   uint64    `db:"audit_sn"`
		EventType string    `db:"event_type"`
		Payload   []byte    `db:"payload"`
		PushRoute string    `db:"push_route"`
	}
	err := r.db.SelectContext(ctx, &rows,
		`SELECT room_id, round_id, action_seq, audit_sn, event_type, payload, push_route
		 FROM game_action_log WHERE round_id=$1 AND action_seq > $2 ORDER BY action_seq`,
		roundID, sinceSeq)
	if err != nil {
		return nil, err
	}
	out := make([]Entry, len(rows))
	for i, row := range rows {
		out[i] = Entry{
			RoomID: row.RoomID, RoundID: row.RoundID, ActionSeq: row.ActionSeq,
			AuditSN: row.AuditSN, EventType: row.EventType, Payload: row.Payload, PushRoute: row.PushRoute,
		}
	}
	return out, nil
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func Now() time.Time { return time.Now().UTC() }
