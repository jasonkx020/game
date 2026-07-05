package club

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

var (
	ErrNotFound           = errors.New("club not found")
	ErrForbidden          = errors.New("forbidden")
	ErrAlreadyMember      = errors.New("already a member")
	ErrInsufficientPool   = errors.New("insufficient club pool balance")
	ErrInsufficientWallet = errors.New("insufficient room card")
)

type Club struct {
	ID          int64  `db:"id" json:"id"`
	Name        string `db:"name" json:"name"`
	OwnerUserID int64  `db:"owner_user_id" json:"owner_user_id"`
	AgentID     *int64 `db:"agent_id" json:"agent_id,omitempty"`
	Status      string `db:"status" json:"status"`
}

type Member struct {
	ClubID   int64  `db:"club_id" json:"club_id"`
	UserID   int64  `db:"user_id" json:"user_id"`
	Role     string `db:"role" json:"role"`
	Status   string `db:"status" json:"status"`
	Nickname string `db:"nickname" json:"nickname"`
	Phone    string `db:"phone" json:"phone"`
}

type Service struct {
	db *sqlx.DB
}

func NewService(db *sqlx.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Create(ctx context.Context, ownerID int64, name string) (*Club, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	var c Club
	err = tx.QueryRowxContext(ctx,
		`INSERT INTO club (name, owner_user_id) VALUES ($1, $2)
		 RETURNING id, name, owner_user_id, agent_id, status`, name, ownerID).StructScan(&c)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO club_member (club_id, user_id, role) VALUES ($1, $2, 'admin')`, c.ID, ownerID); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO club_room_card_pool (club_id, balance) VALUES ($1, 0)`, c.ID); err != nil {
		return nil, err
	}
	return &c, tx.Commit()
}

func (s *Service) Get(ctx context.Context, clubID int64) (*Club, error) {
	var c Club
	if err := s.db.GetContext(ctx, &c,
		`SELECT id, name, owner_user_id, agent_id, status FROM club WHERE id=$1 AND status='active'`, clubID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (s *Service) ListByUser(ctx context.Context, userID int64) ([]Club, error) {
	var clubs []Club
	err := s.db.SelectContext(ctx, &clubs,
		`SELECT c.id, c.name, c.owner_user_id, c.agent_id, c.status
		 FROM club c JOIN club_member m ON c.id = m.club_id
		 WHERE m.user_id=$1 AND m.status='active' AND c.status='active'
		 ORDER BY c.id`, userID)
	return clubs, err
}

func (s *Service) ListAll(ctx context.Context) ([]Club, error) {
	var clubs []Club
	err := s.db.SelectContext(ctx, &clubs,
		`SELECT id, name, owner_user_id, agent_id, status FROM club WHERE status='active' ORDER BY id`)
	return clubs, err
}

func (s *Service) IsAdmin(ctx context.Context, clubID, userID int64) (bool, error) {
	var role string
	err := s.db.GetContext(ctx, &role,
		`SELECT role FROM club_member WHERE club_id=$1 AND user_id=$2 AND status='active'`, clubID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return role == "admin", nil
}

func (s *Service) RequireAdmin(ctx context.Context, clubID, userID int64) error {
	ok, err := s.IsAdmin(ctx, clubID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

func (s *Service) ListMembers(ctx context.Context, clubID int64) ([]Member, error) {
	var members []Member
	err := s.db.SelectContext(ctx, &members,
		`SELECT m.club_id, m.user_id, m.role, m.status, u.nickname, u.phone
		 FROM club_member m JOIN users u ON u.id = m.user_id
		 WHERE m.club_id=$1 AND m.status='active' ORDER BY m.joined_at`, clubID)
	return members, err
}

func (s *Service) AddMember(ctx context.Context, clubID, userID int64) error {
	if _, err := s.Get(ctx, clubID); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO club_member (club_id, user_id, role) VALUES ($1, $2, 'member')
		 ON CONFLICT (club_id, user_id) DO UPDATE SET status='active'`, clubID, userID)
	return err
}

func (s *Service) RemoveMember(ctx context.Context, clubID, targetUserID, actorID int64) error {
	if err := s.RequireAdmin(ctx, clubID, actorID); err != nil {
		return err
	}
	c, err := s.Get(ctx, clubID)
	if err != nil {
		return err
	}
	if c.OwnerUserID == targetUserID {
		return fmt.Errorf("cannot remove club owner")
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE club_member SET status='removed' WHERE club_id=$1 AND user_id=$2`, clubID, targetUserID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) PoolBalance(ctx context.Context, clubID int64) (int64, error) {
	var bal int64
	err := s.db.GetContext(ctx, &bal,
		`SELECT balance FROM club_room_card_pool WHERE club_id=$1`, clubID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	return bal, err
}

func (s *Service) TransferToPool(ctx context.Context, clubID, userID, amount int64, auditSN uint64) (int64, error) {
	if err := s.RequireAdmin(ctx, clubID, userID); err != nil {
		return 0, err
	}
	if amount <= 0 {
		return 0, fmt.Errorf("amount must be positive")
	}

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
	if bal < amount {
		return 0, ErrInsufficientWallet
	}
	newBal := bal - amount
	if _, err := tx.ExecContext(ctx,
		`UPDATE wallet_room_card SET balance=$1, updated_at=now() WHERE user_id=$2`, newBal, userID); err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO wallet_ledger (user_id, wallet_type, delta, balance_after, reason, ref_id, audit_sn)
		 VALUES ($1, 'room_card', $2, $3, 'club_pool_transfer', NULL, $4)`,
		userID, -amount, newBal, auditSN); err != nil {
		return 0, err
	}

	var poolBal int64
	if err := tx.GetContext(ctx, &poolBal,
		`SELECT balance FROM club_room_card_pool WHERE club_id=$1 FOR UPDATE`, clubID); err != nil {
		return 0, err
	}
	poolBal += amount
	if _, err := tx.ExecContext(ctx,
		`UPDATE club_room_card_pool SET balance=$1, updated_at=now() WHERE club_id=$2`, poolBal, clubID); err != nil {
		return 0, err
	}
	return poolBal, tx.Commit()
}

func (s *Service) DeductPool(ctx context.Context, clubID, amount int64) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	var bal int64
	if err := tx.GetContext(ctx, &bal,
		`SELECT balance FROM club_room_card_pool WHERE club_id=$1 FOR UPDATE`, clubID); err != nil {
		return err
	}
	if bal < amount {
		return ErrInsufficientPool
	}
	newBal := bal - amount
	if _, err := tx.ExecContext(ctx,
		`UPDATE club_room_card_pool SET balance=$1, updated_at=now() WHERE club_id=$2`, newBal, clubID); err != nil {
		return err
	}
	return tx.Commit()
}
