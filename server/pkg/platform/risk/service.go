package risk

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

const (
	AlertSameIP      = "same_ip"
	AlertGPSNear     = "gps_near"
	AlertDevice      = "device"
	AlertAbnormalPnL = "abnormal_pnl"
	AlertMinor       = "minor"

	StatusOpen      = "open"
	StatusReviewed  = "reviewed"
	StatusDismissed = "dismissed"

	RiskActive = "active"
	RiskFrozen = "frozen"

	MaxAccountsPerDevice = 2
	MaxSameIPPerTable    = 3
	GPSAlertMeters       = 200.0
	AbnormalPnLMultiple  = 5.0
	MinorDailyMinutes    = 90
	AgentDepositRequired = 5000.0
	WithdrawHoldDays     = 7
)

type Config struct {
	GPSHardBlock bool
	AutoFreeze   bool
}

type Presence struct {
	PlayerID int64    `json:"player_id"`
	IP       string   `json:"ip"`
	DeviceID string   `json:"device_id"`
	Lat      *float64 `json:"lat,omitempty"`
	Lng      *float64 `json:"lng,omitempty"`
}

type Alert struct {
	ID                 int64           `db:"id" json:"id"`
	Type               string          `db:"type" json:"type"`
	Status             string          `db:"status" json:"status"`
	PlayerID           *int64          `db:"player_id" json:"player_id,omitempty"`
	RoomID             *uuid.UUID      `db:"room_id" json:"room_id,omitempty"`
	RoundID            *uuid.UUID      `db:"round_id" json:"round_id,omitempty"`
	SettlementAuditSN  *int64          `db:"settlement_audit_sn" json:"settlement_audit_sn,omitempty"`
	Meta               json.RawMessage `db:"meta" json:"meta"`
	CreatedAt          time.Time       `db:"created_at" json:"created_at"`
	ReviewedBy         *int64          `db:"reviewed_by" json:"reviewed_by,omitempty"`
	ReviewedAt         *time.Time      `db:"reviewed_at" json:"reviewed_at,omitempty"`
	ReviewNote         string          `db:"review_note" json:"review_note"`
}

type Service struct {
	db  *sqlx.DB
	cfg Config
}

func NewService(db *sqlx.DB, cfg Config) *Service {
	return &Service{db: db, cfg: cfg}
}

func (s *Service) GetPlayerRisk(ctx context.Context, playerID int64) (isMinor bool, riskStatus string, err error) {
	err = s.db.QueryRowxContext(ctx,
		`SELECT is_minor, risk_status FROM players WHERE id=$1`, playerID,
	).Scan(&isMinor, &riskStatus)
	if err != nil {
		return false, "", err
	}
	return isMinor, riskStatus, nil
}

func (s *Service) SetPlayerMinor(ctx context.Context, playerID int64, isMinor bool) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE players SET is_minor=$1, updated_at=now() WHERE id=$2`, isMinor, playerID)
	return err
}

func (s *Service) SetRiskStatus(ctx context.Context, playerID int64, status string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE players SET risk_status=$1, updated_at=now() WHERE id=$2`, status, playerID)
	return err
}

// CheckBind enforces device limit, curfew, playtime, frozen.
func (s *Service) CheckBind(ctx context.Context, playerID int64, deviceID, ip, platform string, now time.Time) error {
	isMinor, status, err := s.GetPlayerRisk(ctx, playerID)
	if err != nil {
		return err
	}
	if status == RiskFrozen {
		return ErrFrozen
	}
	if isMinor {
		if inMinorCurfew(now) {
			_ = s.CreateAlert(ctx, AlertMinor, &playerID, nil, nil, nil, map[string]interface{}{
				"reason": "curfew", "hour": now.Hour(),
			})
			return ErrCurfew
		}
		mins, err := s.TodayPlaytime(ctx, playerID, now)
		if err != nil {
			return err
		}
		if mins >= MinorDailyMinutes {
			_ = s.CreateAlert(ctx, AlertMinor, &playerID, nil, nil, nil, map[string]interface{}{
				"reason": "playtime", "minutes": mins,
			})
			return ErrPlaytime
		}
	}
	if deviceID == "" {
		return nil
	}
	n, err := s.CountOtherAccountsOnDevice(ctx, deviceID, playerID)
	if err != nil {
		return err
	}
	if n >= MaxAccountsPerDevice {
		_ = s.CreateAlert(ctx, AlertDevice, &playerID, nil, nil, nil, map[string]interface{}{
			"device_id": deviceID, "other_accounts": n,
		})
		return ErrDeviceLimit
	}
	return s.UpsertDevice(ctx, playerID, deviceID, ip, platform)
}

func inMinorCurfew(now time.Time) bool {
	h := now.Hour()
	return h >= 22 || h < 8
}

func (s *Service) UpsertDevice(ctx context.Context, playerID int64, deviceID, ip, platform string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO player_devices (player_id, device_id, last_ip, platform, last_seen_at)
		VALUES ($1,$2,$3,$4,now())
		ON CONFLICT (device_id, player_id) DO UPDATE
		SET last_ip=EXCLUDED.last_ip, platform=EXCLUDED.platform, last_seen_at=now()`,
		playerID, deviceID, ip, platform)
	return err
}

func (s *Service) CountOtherAccountsOnDevice(ctx context.Context, deviceID string, excludePlayerID int64) (int, error) {
	var n int
	err := s.db.GetContext(ctx, &n,
		`SELECT COUNT(*) FROM player_devices WHERE device_id=$1 AND player_id<>$2`,
		deviceID, excludePlayerID)
	return n, err
}

// CheckJoinTable evaluates same-IP / GPS against peers already seated.
func (s *Service) CheckJoinTable(ctx context.Context, joiner Presence, peers []Presence, roomID uuid.UUID, hardGPS bool) error {
	if joiner.IP != "" {
		same := 0
		for _, p := range peers {
			if p.PlayerID == joiner.PlayerID {
				continue
			}
			if p.IP != "" && p.IP == joiner.IP {
				same++
			}
		}
		if same >= MaxSameIPPerTable {
			pid := joiner.PlayerID
			_ = s.CreateAlert(ctx, AlertSameIP, &pid, &roomID, nil, nil, map[string]interface{}{
				"ip": joiner.IP, "same_count": same,
			})
			return ErrSameIP
		}
	}
	if joiner.Lat != nil && joiner.Lng != nil {
		for _, p := range peers {
			if p.PlayerID == joiner.PlayerID || p.Lat == nil || p.Lng == nil {
				continue
			}
			d := HaversineMeters(*joiner.Lat, *joiner.Lng, *p.Lat, *p.Lng)
			if d < GPSAlertMeters {
				pid := joiner.PlayerID
				_ = s.CreateAlert(ctx, AlertGPSNear, &pid, &roomID, nil, nil, map[string]interface{}{
					"peer_id": p.PlayerID, "meters": d,
				})
				if hardGPS || s.cfg.GPSHardBlock {
					return ErrGPSNear
				}
				break
			}
		}
	}
	return nil
}

func (s *Service) CheckPlayAccess(ctx context.Context, playerID int64, now time.Time) error {
	isMinor, status, err := s.GetPlayerRisk(ctx, playerID)
	if err != nil {
		return err
	}
	if status == RiskFrozen {
		return ErrFrozen
	}
	if !isMinor {
		return nil
	}
	if inMinorCurfew(now) {
		return ErrCurfew
	}
	mins, err := s.TodayPlaytime(ctx, playerID, now)
	if err != nil {
		return err
	}
	if mins >= MinorDailyMinutes {
		return ErrPlaytime
	}
	return nil
}

func (s *Service) CheckRecharge(ctx context.Context, playerID int64) error {
	isMinor, status, err := s.GetPlayerRisk(ctx, playerID)
	if err != nil {
		return err
	}
	if status == RiskFrozen {
		return ErrFrozen
	}
	if isMinor {
		return ErrMinorRecharge
	}
	return nil
}

func (s *Service) TodayPlaytime(ctx context.Context, playerID int64, now time.Time) (int, error) {
	day := now.Format("2006-01-02")
	var mins int
	err := s.db.GetContext(ctx, &mins,
		`SELECT minutes FROM player_playtime_daily WHERE player_id=$1 AND day=$2::date`,
		playerID, day)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return mins, err
}

func (s *Service) AddPlaytime(ctx context.Context, playerID int64, minutes int, now time.Time) error {
	if minutes <= 0 {
		return nil
	}
	day := now.Format("2006-01-02")
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO player_playtime_daily (player_id, day, minutes) VALUES ($1,$2::date,$3)
		ON CONFLICT (player_id, day) DO UPDATE SET minutes = player_playtime_daily.minutes + EXCLUDED.minutes`,
		playerID, day, minutes)
	return err
}

func (s *Service) CreateAlert(ctx context.Context, typ string, playerID *int64, roomID, roundID *uuid.UUID, settleSN *int64, meta map[string]interface{}) error {
	b, _ := json.Marshal(meta)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO risk_alerts (type, status, player_id, room_id, round_id, settlement_audit_sn, meta)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		typ, StatusOpen, playerID, roomID, roundID, settleSN, b)
	return err
}

func (s *Service) ListAlerts(ctx context.Context, typ, status string, page, pageSize int) ([]Alert, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	where := []string{"1=1"}
	args := []interface{}{}
	if typ != "" {
		args = append(args, typ)
		where = append(where, fmt.Sprintf("type=$%d", len(args)))
	}
	if status != "" {
		args = append(args, status)
		where = append(where, fmt.Sprintf("status=$%d", len(args)))
	}
	w := strings.Join(where, " AND ")
	var total int
	if err := s.db.GetContext(ctx, &total, "SELECT COUNT(*) FROM risk_alerts WHERE "+w, args...); err != nil {
		return nil, 0, err
	}
	args = append(args, pageSize, (page-1)*pageSize)
	q := fmt.Sprintf(`SELECT id, type, status, player_id, room_id, round_id, settlement_audit_sn, meta,
		created_at, reviewed_by, reviewed_at, review_note
		FROM risk_alerts WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, w, len(args)-1, len(args))
	var rows []Alert
	if err := s.db.SelectContext(ctx, &rows, q, args...); err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (s *Service) GetAlert(ctx context.Context, id int64) (*Alert, error) {
	var a Alert
	err := s.db.GetContext(ctx, &a, `
		SELECT id, type, status, player_id, room_id, round_id, settlement_audit_sn, meta,
			created_at, reviewed_by, reviewed_at, review_note
		FROM risk_alerts WHERE id=$1`, id)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Service) ReviewAlert(ctx context.Context, id, reviewerID int64, decision, note string, freeze bool) error {
	a, err := s.GetAlert(ctx, id)
	if err != nil {
		return err
	}
	status := StatusDismissed
	if decision == "confirmed" {
		status = StatusReviewed
	}
	_, err = s.db.ExecContext(ctx, `
		UPDATE risk_alerts SET status=$1, reviewed_by=$2, reviewed_at=now(), review_note=$3 WHERE id=$4`,
		status, reviewerID, note, id)
	if err != nil {
		return err
	}
	if freeze && decision == "confirmed" && a.PlayerID != nil {
		return s.SetRiskStatus(ctx, *a.PlayerID, RiskFrozen)
	}
	return nil
}

// EvaluateSettlement flags abnormal absolute coin/score deltas vs 24h mean for the player.
func (s *Service) EvaluateSettlement(ctx context.Context, playerID int64, absDelta float64, roomID, roundID uuid.UUID, settleSN int64) error {
	if absDelta <= 0 {
		return nil
	}
	var avg sql.NullFloat64
	err := s.db.GetContext(ctx, &avg, `
		SELECT AVG(ABS(delta)) FROM wallet_ledger
		WHERE reason='settlement' AND created_at > now() - interval '24 hours' AND user_id=$1`, playerID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	mean := 0.0
	if avg.Valid {
		mean = avg.Float64
	}
	if mean <= 0 {
		return nil // insufficient history
	}
	if absDelta < mean*AbnormalPnLMultiple {
		return nil
	}
	pid := playerID
	rid := roomID
	rnd := roundID
	sn := settleSN
	meta := map[string]interface{}{
		"abs_delta": absDelta, "mean_24h": mean, "multiple": absDelta / mean,
	}
	if err := s.CreateAlert(ctx, AlertAbnormalPnL, &pid, &rid, &rnd, &sn, meta); err != nil {
		return err
	}
	if s.cfg.AutoFreeze {
		return s.SetRiskStatus(ctx, playerID, RiskFrozen)
	}
	return nil
}

// --- Agent deposit / withdraw ---

type DepositRow struct {
	ID           int64     `db:"id" json:"id"`
	AgentAdminID int64     `db:"agent_admin_id" json:"agent_admin_id"`
	Kind         string    `db:"kind" json:"kind"`
	AmountCNY    float64   `db:"amount_cny" json:"amount_cny"`
	BalanceAfter float64   `db:"balance_after" json:"balance_after"`
	Reason       string    `db:"reason" json:"reason"`
	ActorAdminID *int64    `db:"actor_admin_id" json:"actor_admin_id,omitempty"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

type WithdrawalRow struct {
	ID           int64      `db:"id" json:"id"`
	AgentAdminID int64      `db:"agent_admin_id" json:"agent_admin_id"`
	AmountCNY    float64    `db:"amount_cny" json:"amount_cny"`
	Status       string     `db:"status" json:"status"`
	RequestedAt  time.Time  `db:"requested_at" json:"requested_at"`
	EligibleAt   time.Time  `db:"eligible_at" json:"eligible_at"`
	ReviewedBy   *int64     `db:"reviewed_by" json:"reviewed_by,omitempty"`
	ReviewedAt   *time.Time `db:"reviewed_at" json:"reviewed_at,omitempty"`
	ReviewNote   string     `db:"review_note" json:"review_note"`
	ForceApprove bool       `db:"force_approve" json:"force_approve"`
}

func (s *Service) AgentDepositBalance(ctx context.Context, agentID int64) (float64, error) {
	var bal sql.NullFloat64
	err := s.db.GetContext(ctx, &bal, `
		SELECT balance_after FROM agent_deposits WHERE agent_admin_id=$1 ORDER BY id DESC LIMIT 1`, agentID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if !bal.Valid {
		return 0, nil
	}
	return bal.Float64, nil
}

func (s *Service) RecordAgentDeposit(ctx context.Context, agentID int64, kind string, amount float64, reason string, actorID *int64) (*DepositRow, error) {
	if kind != "deposit" && kind != "penalty" {
		return nil, fmt.Errorf("%w: kind", ErrBadRequest)
	}
	if amount <= 0 {
		return nil, fmt.Errorf("%w: amount", ErrBadRequest)
	}
	cur, err := s.AgentDepositBalance(ctx, agentID)
	if err != nil {
		return nil, err
	}
	next := cur + amount
	if kind == "penalty" {
		next = cur - amount
	}
	var row DepositRow
	err = s.db.QueryRowxContext(ctx, `
		INSERT INTO agent_deposits (agent_admin_id, kind, amount_cny, balance_after, reason, actor_admin_id)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, agent_admin_id, kind, amount_cny, balance_after, reason, actor_admin_id, created_at`,
		agentID, kind, amount, next, reason, actorID).StructScan(&row)
	return &row, err
}

func (s *Service) ListAgentDeposits(ctx context.Context, agentID int64, limit int) ([]DepositRow, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows []DepositRow
	err := s.db.SelectContext(ctx, &rows, `
		SELECT id, agent_admin_id, kind, amount_cny, balance_after, reason, actor_admin_id, created_at
		FROM agent_deposits WHERE agent_admin_id=$1 ORDER BY id DESC LIMIT $2`, agentID, limit)
	return rows, err
}

func (s *Service) RequestWithdrawal(ctx context.Context, agentID int64, amount float64, now time.Time) (*WithdrawalRow, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("%w: amount", ErrBadRequest)
	}
	eligible := now.Add(time.Duration(WithdrawHoldDays) * 24 * time.Hour)
	var row WithdrawalRow
	err := s.db.QueryRowxContext(ctx, `
		INSERT INTO agent_withdrawals (agent_admin_id, amount_cny, status, requested_at, eligible_at)
		VALUES ($1,$2,'pending',$3,$4)
		RETURNING id, agent_admin_id, amount_cny, status, requested_at, eligible_at, reviewed_by, reviewed_at, review_note, force_approve`,
		agentID, amount, now, eligible).StructScan(&row)
	return &row, err
}

func (s *Service) ListWithdrawals(ctx context.Context, agentID int64, status string, limit int) ([]WithdrawalRow, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows []WithdrawalRow
	if status == "" {
		err := s.db.SelectContext(ctx, &rows, `
			SELECT id, agent_admin_id, amount_cny, status, requested_at, eligible_at, reviewed_by, reviewed_at, review_note, force_approve
			FROM agent_withdrawals WHERE agent_admin_id=$1 ORDER BY id DESC LIMIT $2`, agentID, limit)
		return rows, err
	}
	err := s.db.SelectContext(ctx, &rows, `
		SELECT id, agent_admin_id, amount_cny, status, requested_at, eligible_at, reviewed_by, reviewed_at, review_note, force_approve
		FROM agent_withdrawals WHERE agent_admin_id=$1 AND status=$2 ORDER BY id DESC LIMIT $3`, agentID, status, limit)
	return rows, err
}

func (s *Service) ReviewWithdrawal(ctx context.Context, id, reviewerID int64, approve, force bool, note string, now time.Time) error {
	var row WithdrawalRow
	err := s.db.GetContext(ctx, &row, `
		SELECT id, agent_admin_id, amount_cny, status, requested_at, eligible_at, reviewed_by, reviewed_at, review_note, force_approve
		FROM agent_withdrawals WHERE id=$1`, id)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if row.Status != "pending" {
		return fmt.Errorf("%w: already reviewed", ErrBadRequest)
	}
	status := "rejected"
	if approve {
		if !force && now.Before(row.EligibleAt) {
			return fmt.Errorf("%w: not eligible until T+%d", ErrForbidden, WithdrawHoldDays)
		}
		status = "approved"
	}
	_, err = s.db.ExecContext(ctx, `
		UPDATE agent_withdrawals SET status=$1, reviewed_by=$2, reviewed_at=$3, review_note=$4, force_approve=$5 WHERE id=$6`,
		status, reviewerID, now, note, force && approve, id)
	return err
}

// TableSnapshot returns last-known presence for players who joined a room
// (from room_event_log JOIN payloads + player_devices).
func (s *Service) TableSnapshot(ctx context.Context, roomID uuid.UUID) ([]Presence, error) {
	type row struct {
		UserID   int64           `db:"user_id"`
		Payload  json.RawMessage `db:"payload"`
		DeviceID sql.NullString  `db:"device_id"`
		LastIP   sql.NullString  `db:"last_ip"`
	}
	var rows []row
	err := s.db.SelectContext(ctx, &rows, `
		SELECT DISTINCT ON (e.user_id) e.user_id, e.payload, d.device_id, d.last_ip
		FROM room_event_log e
		LEFT JOIN LATERAL (
			SELECT device_id, last_ip FROM player_devices
			WHERE player_id = e.user_id ORDER BY last_seen_at DESC LIMIT 1
		) d ON true
		WHERE e.room_id=$1 AND e.event_type IN ('JOIN','JOIN_SPECTATOR')
		ORDER BY e.user_id, e.room_seq DESC`, roomID)
	if err != nil {
		return nil, err
	}
	out := make([]Presence, 0, len(rows))
	for _, r := range rows {
		p := Presence{PlayerID: r.UserID}
		if r.LastIP.Valid {
			p.IP = r.LastIP.String
		}
		if r.DeviceID.Valid {
			p.DeviceID = r.DeviceID.String
		}
		var meta map[string]interface{}
		if len(r.Payload) > 0 {
			_ = json.Unmarshal(r.Payload, &meta)
		}
		if meta != nil {
			if ip, ok := meta["ip"].(string); ok && ip != "" {
				p.IP = ip
			}
			if d, ok := meta["device_id"].(string); ok && d != "" {
				p.DeviceID = d
			}
			if lat, ok := asFloat(meta["lat"]); ok {
				p.Lat = &lat
			}
			if lng, ok := asFloat(meta["lng"]); ok {
				p.Lng = &lng
			}
		}
		out = append(out, p)
	}
	return out, nil
}

func asFloat(v interface{}) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case json.Number:
		f, err := t.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

// AbsMax helper for settlement evaluation.
func AbsMax(vals []float64) float64 {
	m := 0.0
	for _, v := range vals {
		a := math.Abs(v)
		if a > m {
			m = a
		}
	}
	return m
}
