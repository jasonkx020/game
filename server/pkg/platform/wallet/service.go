package wallet

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
)

var ErrInsufficientBalance = errors.New("insufficient balance")
var ErrUnknownProduct = errors.New("unknown product")

type Product struct {
	ID        string
	AmountCNY int
	Cards     int64
}

var Products = map[string]Product{
	"rc_10":  {ID: "rc_10", AmountCNY: 6, Cards: 10},
	"rc_50":  {ID: "rc_50", AmountCNY: 28, Cards: 50},
	"rc_200": {ID: "rc_200", AmountCNY: 98, Cards: 200},
}

type RechargeOrder struct {
	ID        int64  `db:"id" json:"id"`
	UserID    int64  `db:"user_id" json:"user_id"`
	ProductID string `db:"product_id" json:"product_id"`
	AmountCNY int    `db:"amount_cny" json:"amount_cny"`
	Cards     int64  `db:"cards" json:"cards"`
	AuditSN   int64  `db:"audit_sn" json:"audit_sn"`
	CreatedAt string `db:"created_at" json:"created_at"`
}

type Service struct {
	db *sqlx.DB
}

func NewService(db *sqlx.DB) *Service {
	return &Service{db: db}
}

func (s *Service) RoomCardBalance(ctx context.Context, userID int64) (int64, error) {
	var bal int64
	err := s.db.GetContext(ctx, &bal, `SELECT balance FROM wallet_room_card WHERE user_id=$1`, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return bal, err
}

func (s *Service) GameCoinBalance(ctx context.Context, userID int64, gameID string) (int64, error) {
	var bal int64
	err := s.db.GetContext(ctx, &bal,
		`SELECT balance FROM wallet_game_coin WHERE user_id=$1 AND game_id=$2`, userID, gameID)
	if errors.Is(err, sql.ErrNoRows) {
		// 未开户的游戏金币按 0 返回，避免大厅列表因缺行 500
		return 0, nil
	}
	return bal, err
}

func (s *Service) CreditRoomCard(ctx context.Context, userID, amount int64, reason, refID string, auditSN uint64) (int64, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback() //nolint:errcheck

	var bal int64
	if err := tx.GetContext(ctx, &bal,
		`SELECT balance FROM wallet_room_card WHERE user_id=$1 FOR UPDATE`, userID); err != nil {
		return 0, err
	}
	newBal := bal + amount
	if _, err := tx.ExecContext(ctx,
		`UPDATE wallet_room_card SET balance=$1, updated_at=now() WHERE user_id=$2`, newBal, userID); err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO wallet_ledger (user_id, wallet_type, delta, balance_after, reason, ref_id, audit_sn)
		 VALUES ($1, 'room_card', $2, $3, $4, $5::uuid, $6)`,
		userID, amount, newBal, reason, nullUUID(refID), auditSN); err != nil {
		return 0, err
	}
	return newBal, tx.Commit()
}

func (s *Service) MockRecharge(ctx context.Context, userID int64, productID string, auditSN uint64) (*RechargeOrder, int64, error) {
	p, ok := Products[productID]
	if !ok {
		return nil, 0, ErrUnknownProduct
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback() //nolint:errcheck

	var order RechargeOrder
	err = tx.QueryRowxContext(ctx,
		`INSERT INTO recharge_order (user_id, product_id, amount_cny, cards, audit_sn)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, user_id, product_id, amount_cny, cards, audit_sn, created_at::text`,
		userID, p.ID, p.AmountCNY, p.Cards, auditSN).StructScan(&order)
	if err != nil {
		return nil, 0, err
	}

	var bal int64
	if err := tx.GetContext(ctx, &bal,
		`SELECT balance FROM wallet_room_card WHERE user_id=$1 FOR UPDATE`, userID); err != nil {
		return nil, 0, err
	}
	newBal := bal + p.Cards
	if _, err := tx.ExecContext(ctx,
		`UPDATE wallet_room_card SET balance=$1, updated_at=now() WHERE user_id=$2`, newBal, userID); err != nil {
		return nil, 0, err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO wallet_ledger (user_id, wallet_type, delta, balance_after, reason, ref_id, audit_sn)
		 VALUES ($1, 'room_card', $2, $3, 'recharge', NULL, $4)`,
		userID, p.Cards, newBal, auditSN); err != nil {
		return nil, 0, err
	}

	return &order, newBal, tx.Commit()
}

func (s *Service) RechargeHistory(ctx context.Context, userID int64, limit int) ([]RechargeOrder, error) {
	if limit <= 0 {
		limit = 20
	}
	var orders []RechargeOrder
	err := s.db.SelectContext(ctx, &orders,
		`SELECT id, user_id, product_id, amount_cny, cards, audit_sn, created_at::text
		 FROM recharge_order WHERE user_id=$1 ORDER BY created_at DESC LIMIT $2`, userID, limit)
	return orders, err
}

func (s *Service) DeductRoomCard(ctx context.Context, userID int64, amount int64, reason, refID string, auditSN uint64) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	var bal int64
	if err := tx.GetContext(ctx, &bal,
		`SELECT balance FROM wallet_room_card WHERE user_id=$1 FOR UPDATE`, userID); err != nil {
		return err
	}
	if bal < amount {
		return ErrInsufficientBalance
	}
	newBal := bal - amount
	if _, err := tx.ExecContext(ctx,
		`UPDATE wallet_room_card SET balance=$1, updated_at=now() WHERE user_id=$2`, newBal, userID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO wallet_ledger (user_id, wallet_type, delta, balance_after, reason, ref_id, audit_sn)
		 VALUES ($1, 'room_card', $2, $3, $4, $5::uuid, $6)`,
		userID, -amount, newBal, reason, nullUUID(refID), auditSN); err != nil {
		return err
	}
	return tx.Commit()
}

// ApplyGameCoinDelta adjusts game coin within an existing transaction (may be negative).
// Creates the wallet row if missing. balance is floored at 0 for deductions beyond balance
// only when allowNegative is false; otherwise returns ErrInsufficientBalance.
func (s *Service) ApplyGameCoinDeltaTx(ctx context.Context, tx *sqlx.Tx, userID int64, gameID string, delta int64, reason, refID string, auditSN uint64) (int64, error) {
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO wallet_game_coin (user_id, game_id, balance) VALUES ($1,$2,0)
		 ON CONFLICT (user_id, game_id) DO NOTHING`, userID, gameID); err != nil {
		return 0, err
	}
	var bal int64
	if err := tx.GetContext(ctx, &bal,
		`SELECT balance FROM wallet_game_coin WHERE user_id=$1 AND game_id=$2 FOR UPDATE`, userID, gameID); err != nil {
		return 0, err
	}
	newBal := bal + delta
	if newBal < 0 {
		return 0, ErrInsufficientBalance
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE wallet_game_coin SET balance=$1, updated_at=now() WHERE user_id=$2 AND game_id=$3`,
		newBal, userID, gameID); err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO wallet_ledger (user_id, wallet_type, game_id, delta, balance_after, reason, ref_id, audit_sn)
		 VALUES ($1, 'game_coin', $2, $3, $4, $5, $6::uuid, $7)`,
		userID, gameID, delta, newBal, reason, nullUUID(refID), auditSN); err != nil {
		return 0, err
	}
	return newBal, nil
}

func (s *Service) CreditGameCoin(ctx context.Context, userID int64, gameID string, amount int64, reason, refID string, auditSN uint64) (int64, error) {
	if amount < 0 {
		return 0, errors.New("credit amount must be non-negative")
	}
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback() //nolint:errcheck
	bal, err := s.ApplyGameCoinDeltaTx(ctx, tx, userID, gameID, amount, reason, refID, auditSN)
	if err != nil {
		return 0, err
	}
	return bal, tx.Commit()
}

func (s *Service) DeductGameCoin(ctx context.Context, userID int64, gameID string, amount int64, reason, refID string, auditSN uint64) error {
	if amount < 0 {
		return errors.New("deduct amount must be non-negative")
	}
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	if _, err := s.ApplyGameCoinDeltaTx(ctx, tx, userID, gameID, -amount, reason, refID, auditSN); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) BeginTx(ctx context.Context) (*sqlx.Tx, error) {
	return s.db.BeginTxx(ctx, nil)
}

func nullUUID(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
